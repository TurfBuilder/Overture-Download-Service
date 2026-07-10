package downloader

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nuid"
	"github.com/paulmach/orb/geojson"
)

// SelectionParameters is the JSON payload of a download request.
type SelectionParameters struct {
	// Area is a GeoJSON Polygon (or MultiPolygon); all businesses within it
	// are fetched from Overture.
	Area *geojson.Geometry `json:"area"`
}

// worker pulls one request at a time from the consumer until ctx is cancelled.
// The pool size is the concurrency limit: a worker only fetches again once the
// previous request is fully handled.
func worker(
	ctx context.Context,
	waitGroup *sync.WaitGroup,
	consumer jetstream.Consumer,
	jwks keyfunc.Keyfunc,
	jobs *JobStore,
	store *ObjectStore,
	overtureRelease string,
	debug bool,
	id int,
) {
	defer waitGroup.Done()

	for {
		if ctx.Err() != nil {
			logger.Infow("worker stopping", "worker", id)
			return
		}

		// Wait up to 5s for a message, then loop so cancellation is noticed.
		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
		if err != nil {
			logger.Errorw("fetch failed", "worker", id, "error", err)
			time.Sleep(time.Second) // avoid a hot loop while NATS is unreachable
			continue
		}

		for msg := range msgs.Messages() {
			handleRequest(ctx, msg, jwks, jobs, store, overtureRelease, debug, id)
		}
	}
}

func handleRequest(ctx context.Context, msg jetstream.Msg, jwks keyfunc.Keyfunc, jobs *JobStore, store *ObjectStore, overtureRelease string, debug bool, id int) {
	// Every request gets acked: bad ones must not be redelivered forever, and
	// good ones are done. Redelivery-on-crash still works because the ack only
	// happens after handling.
	defer msg.Ack()

	if debug {
		logger.Warnw("debug mode: skipping access token validation", "worker", id)
	} else {
		accessToken := msg.Headers().Get("Access-Token")
		if !isAccessTokenValid(accessToken, jwks) {
			logger.Warnw("rejected request with invalid access token", "worker", id)
			return
		}
	}

	jobID := msg.Headers().Get("Job-Id")
	if jobID == "" {
		jobID = nuid.Next() // no id supplied by the requester; generate one
	}

	job, err := jobs.Start(ctx, jobID)
	if err != nil {
		logger.Errorw("failed to record job start", "worker", id, "job", jobID, "error", err)
	}

	fail := func(reason string, err error) {
		logger.Errorw(reason, "worker", id, "job", jobID, "error", err)
		job.Error = fmt.Sprintf("%s: %v", reason, err)
		if err := jobs.Finish(ctx, job, JobFailed); err != nil {
			logger.Errorw("failed to record job failure", "worker", id, "job", jobID, "error", err)
		}
	}

	// Parse and validate the request payload
	var params SelectionParameters
	if err := json.Unmarshal(msg.Data(), &params); err != nil {
		fail("invalid request payload", err)
		return
	}
	if params.Area == nil {
		fail("invalid request payload", fmt.Errorf("missing required parameter: area"))
		return
	}
	switch params.Area.Type {
	case "Polygon", "MultiPolygon":
	default:
		fail("invalid request payload", fmt.Errorf("area must be a Polygon or MultiPolygon, got %s", params.Area.Type))
		return
	}

	logger.Infow("processing download request", "worker", id, "job", jobID)

	// Query Overture for all places inside the requested area
	places, err := fetchPlacesWithin(ctx, overtureRelease, params.Area)
	if err != nil {
		fail("overture query failed", err)
		return
	}

	// Upload the result set and record where it landed
	data, err := placesToCSV(places)
	if err != nil {
		fail("encoding result failed", err)
		return
	}

	resultURL, err := store.UploadCSV(ctx, fmt.Sprintf("results/%s.csv", jobID), data)
	if err != nil {
		fail("uploading result failed", err)
		return
	}

	job.ResultUrl = resultURL
	if err := jobs.Finish(ctx, job, JobCompleted); err != nil {
		logger.Errorw("failed to record job completion", "worker", id, "job", jobID, "error", err)
	}

	logger.Infow("job completed", "worker", id, "job", jobID, "places", len(places), "result", resultURL)
}

// placesToCSV renders the result rows in the importer's expected column order.
func placesToCSV(places []Place) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	rows := [][]string{{
		"name", "address_line_1", "address_line_2", "city",
		"state_or_region", "postal_code", "country_code",
		"latitude", "longitude",
	}}
	for _, p := range places {
		rows = append(rows, []string{
			p.Name, p.AddressLine1, p.AddressLine2, p.City,
			p.StateOrRegion, p.PostalCode, p.CountryCode,
			strconv.FormatFloat(p.Latitude, 'f', -1, 64),
			strconv.FormatFloat(p.Longitude, 'f', -1, 64),
		})
	}

	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
