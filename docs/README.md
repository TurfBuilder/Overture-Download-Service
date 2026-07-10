# Overture Downloader Documentation

Overture Downloader is a background service that fetches business/point-of-interest
data from [Overture Maps](https://overturemaps.org/) for a requested geographic
area and uploads the result to an S3-compatible bucket (DigitalOcean Spaces).

Requests arrive as NATS JetStream messages, run asynchronously on a worker pool,
and report their progress through a job record that can be checked over a small
REST API.

- [Getting Started](getting-started.md) — prerequisites, running locally, Docker
- [Configuration](configuration.md) — command-line flags and environment variables
- [Usage](usage.md) — sending requests, job lifecycle, checking status
