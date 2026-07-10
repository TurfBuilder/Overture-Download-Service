# Configuration

The service is configured with command-line flags and environment variables.
Flags control behavior; environment variables carry credentials.

## Command-line flags

| Flag | Default | Description |
|---|---|---|
| `--host` | `127.0.0.1` | NATS server host to connect to. |
| `--port` | `4222` | NATS server port. |
| `--workers` | `1` | Maximum number of download jobs processed at the same time. Requests beyond this wait in the stream until a worker is free. Must be at least 1. |
| `--jwks` | `https://turfbuilder.org/.well-known/jwks.json` | URL of the JSON Web Key Set used to verify request access tokens. Fetched once at startup and refreshed automatically. |
| `--api-port` | `23363` | Port the REST API listens on. |
| `--overture-release` | `2026-06-17.0` | Overture Maps data release to query. Overture publishes new releases regularly **and removes old ones from its bucket**, so a stale value eventually fails with "No files found"; see the [release list](https://docs.overturemaps.org/release/) and keep this current. |
| `--debug` | `false` | Disables JWT validation — every request is accepted without a token. Also skips the JWKS fetch at startup. **Never use in production.** A warning is logged at startup and for every message handled while enabled. |

Example:

```
./overture-downloader --host nats.internal --workers 4 --overture-release 2026-06-17.0
```

## Environment variables (all required)

These configure the S3-compatible object storage (DigitalOcean Spaces) where
finished results are uploaded. The service refuses to start if any is missing.

| Variable | Example | Description |
|---|---|---|
| `S3_ENDPOINT` | `nyc3.digitaloceanspaces.com` | The Spaces/S3 endpoint hostname, without `https://` and without the bucket name. |
| `S3_BUCKET` | `my-results-bucket` | Bucket that receives result files. |
| `S3_ACCESS_KEY` | `DO00...` | Spaces access key. |
| `S3_SECRET_KEY` | *(secret)* | Spaces secret key. |

Results are written to `results/<job-id>.csv` inside the bucket with
content type `text/csv`. See [Usage — The result file](usage.md#the-result-file)
for the column layout.

## Sizing notes

- Each running job can hold roughly 1–2 GB of memory and download hundreds of
  MB to ~2 GB from Overture's public bucket, depending on the size of the
  requested area. Budget memory as roughly `--workers × 2 GB` at peak.
- On first use, the embedded DuckDB engine downloads its `spatial` and `httpfs`
  extensions (tens of MB). This needs outbound internet access and a writable
  home directory; the provided Dockerfile handles both. The first query after
  startup is therefore slower than the rest.
