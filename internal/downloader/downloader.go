package downloader

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

// Config carries everything Run needs; the cmd entrypoint fills it from flags.
type Config struct {
	NatsHost        string
	NatsPort        int
	MaxWorkers      uint
	JwksUrl         string
	ApiPort         int
	OvertureRelease string
	Debug           bool // disables JWT validation; never use in production
}

// Run starts the downloader service and blocks until SIGINT/SIGTERM.
func Run(cfg Config) {

	// Setup the logger
	logger = newLogger()
	defer logger.Sync()

	// Root context: cancelled on Ctrl-C or SIGTERM, which signals every worker to stop
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Setup Security
	var jwks keyfunc.Keyfunc
	if cfg.Debug {
		logger.Warn("DEBUG MODE: JWT validation is disabled, all requests are accepted")
	} else {
		var err error
		jwks, err = fetchJWKS(cfg.JwksUrl)
		if err != nil {
			logger.Fatalw("failed to fetch JWKS", "error", err)
		}
	}

	// Where finished results get uploaded (DigitalOcean Spaces, S3-compatible)
	store, err := newObjectStoreFromEnv()
	if err != nil {
		logger.Fatalw("failed to set up object storage", "error", err)
	}

	// Connect to the NATS server and set up JetStream
	logger.Infof("connecting to %s:%d", cfg.NatsHost, cfg.NatsPort)
	natsConnection, js, err := connectToStream(cfg.NatsHost, cfg.NatsPort)
	if err != nil {
		logger.Fatal(err)
	}
	defer natsConnection.Close()
	logger.Info("Connected to NATS server!")

	// Ensure the stream exists, then attach a durable pull consumer to it
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "DOWNLOADS",
		Subjects: []string{"downloads.requests"},
	})
	if err != nil {
		logger.Fatalw("failed to create stream", "error", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "overture-downloader",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		logger.Fatalw("failed to create consumer", "error", err)
	}

	// Job status records live in a JetStream Key-Value bucket
	jobs, err := newJobStore(ctx, js)
	if err != nil {
		logger.Fatalw("failed to create job store", "error", err)
	}

	// Start the worker pool; the pool size caps how many requests run at once
	var waitGroup sync.WaitGroup
	for i := 0; i < int(cfg.MaxWorkers); i++ {
		waitGroup.Add(1)
		go worker(ctx, &waitGroup, consumer, jwks, jobs, store, cfg.OvertureRelease, cfg.Debug, i)
	}
	logger.Infow("workers started", "count", cfg.MaxWorkers)

	// Serve the REST API for job status lookups
	apiServer := startAPI(fmt.Sprintf(":%d", cfg.ApiPort), jobs)

	// Block until a shutdown signal arrives, then wait for workers to finish
	<-ctx.Done()
	logger.Info("shutting down, waiting for workers to finish")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorw("http server shutdown failed", "error", err)
	}

	waitGroup.Wait()
}
