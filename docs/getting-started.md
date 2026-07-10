# Getting Started

## Prerequisites

- A [NATS server](https://docs.nats.io/) with JetStream enabled (`nats-server -js`).
- A DigitalOcean Spaces bucket (or any S3-compatible bucket) and its credentials.
- For authenticated use: a JWKS endpoint serving the public keys that verify
  request tokens (see [Usage — Authentication](usage.md#authentication)).
- Go 1.25+ to build from source, or Docker.

## Running locally

```sh
# 1. Start NATS with JetStream
nats-server -js

# 2. Provide storage credentials
export S3_ENDPOINT=nyc3.digitaloceanspaces.com
export S3_BUCKET=my-results-bucket
export S3_ACCESS_KEY=...
export S3_SECRET_KEY=...

# 3. Build and run (debug mode skips JWT auth — local testing only)
go build -o overture-downloader ./cmd/overture-downloader
./overture-downloader --debug=true
```

Alternatively, put the four `S3_*` variables in a `.env` file at the repo root
(it's git- and Docker-ignored) and use `./run-dev.sh`, which exports them and
runs the service via `go run`. Arguments are passed through:

```sh
./run-dev.sh --debug=true
```

On startup the service connects to NATS, creates the `DOWNLOADS` stream and its
consumer if they don't exist, creates the `jobs` key-value bucket, starts the
worker pool, and begins serving the REST API (default port 23363).

Stop it with Ctrl-C: the service finishes in-flight jobs before exiting.

## Running with Docker

```sh
docker build -t overture-downloader .

docker run --rm \
  -e S3_ENDPOINT=nyc3.digitaloceanspaces.com \
  -e S3_BUCKET=my-results-bucket \
  -e S3_ACCESS_KEY=... \
  -e S3_SECRET_KEY=... \
  -p 23363:23363 \
  overture-downloader --host my-nats-host --workers 2
```

The image is distroless (no shell inside). Flags go after the image name;
environment variables are passed with `-e`.

## Project layout

```
cmd/overture-downloader/   entrypoint: parses flags, starts the service
internal/downloader/       all application code
docs/                      this documentation
```
