package downloader

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type JobStatus string

const (
	JobRunning   JobStatus = "RUNNING"
	JobFailed    JobStatus = "FAILED"
	JobCompleted JobStatus = "COMPLETED"
)

type Job struct {
	Id          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"` // nil while the job is running
	ResultUrl   string     `json:"result_url,omitempty"`   // set once COMPLETED
	Error       string     `json:"error,omitempty"`        // set when FAILED
}

// JobStore keeps one Job record per job id in a JetStream Key-Value bucket.
type JobStore struct {
	kv jetstream.KeyValue
}

func newJobStore(ctx context.Context, js jetstream.JetStream) (*JobStore, error) {
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "jobs",
		Description: "Overture download job statuses",
	})
	if err != nil {
		return nil, err
	}
	return &JobStore{kv: kv}, nil
}

// Start records a new job as RUNNING and returns it.
func (s *JobStore) Start(ctx context.Context, id string) (Job, error) {
	job := Job{
		Id:        id,
		Status:    JobRunning,
		StartedAt: time.Now().UTC(),
	}
	return job, s.put(ctx, job)
}

// Finish marks the job COMPLETED or FAILED and stamps the completion time.
func (s *JobStore) Finish(ctx context.Context, job Job, status JobStatus) error {
	now := time.Now().UTC()
	job.Status = status
	job.CompletedAt = &now
	return s.put(ctx, job)
}

// Get looks up a job by id. Returns jetstream.ErrKeyNotFound if it doesn't exist.
func (s *JobStore) Get(ctx context.Context, id string) (Job, error) {
	entry, err := s.kv.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	var job Job
	err = json.Unmarshal(entry.Value(), &job)
	return job, err
}

func (s *JobStore) put(ctx context.Context, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = s.kv.Put(ctx, job.Id, data)
	return err
}
