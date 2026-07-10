package downloader

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nats-io/nats.go/jetstream"
)

// startAPI serves the REST endpoints in a background goroutine and returns the
// server so the caller can shut it down gracefully.
func startAPI(addr string, jobs *JobStore) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		job, err := jobs.Get(r.Context(), r.PathValue("id"))
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if err != nil {
			logger.Errorw("job lookup failed", "job", r.PathValue("id"), "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	})

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		logger.Infow("REST API listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorw("http server failed", "error", err)
		}
	}()

	return server
}
