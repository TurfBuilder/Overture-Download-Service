# Usage

## Sending a download request

Publish a message to the JetStream subject `downloads.requests`.

**Payload** — a JSON object with a single parameter:

| Field | Type | Description |
|---|---|---|
| `area` | GeoJSON `Polygon` or `MultiPolygon` | The shape to search. All Overture places (businesses / points of interest) inside it are returned. Coordinates are `[longitude, latitude]`, per the GeoJSON standard. |

**Headers:**

| Header | Required | Description |
|---|---|---|
| `Access-Token` | Yes (unless `--debug`) | A JWT authorizing the request. See [Authentication](#authentication). |
| `Job-Id` | No | Your own id for tracking the job. If omitted, the service generates one (visible in its logs). |

### Example requests

All examples use the [NATS CLI](https://github.com/nats-io/natscli). A basic
request — all of Philadelphia, as a bounding-box `Polygon` (note the first and
last coordinates are the same; GeoJSON rings must close):

```sh
nats pub downloads.requests \
  '{"area":{"type":"Polygon","coordinates":[[[-75.28,39.87],[-74.96,39.87],[-74.96,40.14],[-75.28,40.14],[-75.28,39.87]]]}}' \
  -H "Access-Token:eyJhb..." \
  -H "Job-Id:my-job-123"
```

Larger shapes are easier to keep in a file. With the payload in
`request.json`, pipe it in over stdin:

```sh
nats pub downloads.requests --force-stdin \
  -H "Access-Token:eyJhb..." \
  -H "Job-Id:my-job-124" < request.json
```

Several disjoint areas in one job, as a `MultiPolygon` (against a server
running with `--debug`, so no `Access-Token`; without a `Job-Id` the service
generates one, visible in its logs):

```sh
nats pub downloads.requests \
  '{"area":{"type":"MultiPolygon","coordinates":[
    [[[-75.17,39.94],[-75.14,39.94],[-75.14,39.96],[-75.17,39.96],[-75.17,39.94]]],
    [[[-75.21,39.94],[-75.19,39.94],[-75.19,39.96],[-75.21,39.96],[-75.21,39.94]]]
  ]}}'
```

## Job lifecycle

Each request becomes a job stored in the `jobs` JetStream key-value bucket,
keyed by job id:

1. `RUNNING` — a worker picked the request up; `started_at` is set.
2. `COMPLETED` — the result was uploaded; `completed_at` and `result_url` are set.
3. `FAILED` — something went wrong; `completed_at` and `error` (the reason) are set.

A failed job is **not retried automatically** — inspect the `error`, fix the
request, and send it again. Requests with an invalid access token are rejected
before a job record is created.

If all workers are busy, requests simply wait in the stream — nothing is lost.

## Checking job status

**REST API:**

```
GET http://<host>:23363/jobs/<job-id>
```

Responses:

- `200` — the job record:
  ```json
  {
    "id": "my-job-123",
    "status": "COMPLETED",
    "started_at": "2026-07-09T15:04:05Z",
    "completed_at": "2026-07-09T15:06:41Z",
    "result_url": "https://my-results-bucket.nyc3.digitaloceanspaces.com/results/my-job-123.csv"
  }
  ```
- `404` — no job with that id.

Note: this endpoint is currently unauthenticated. Don't expose the API port to
untrusted networks.

**NATS CLI** (reads the same record): `nats kv get jobs <job-id>`, or watch it
live with `nats kv watch jobs <job-id>`.

## The result file

A CSV at `result_url`, one row per place, with a header row and these columns:

| Column | Description |
|---|---|
| `name` | The place's primary name. |
| `address_line_1` | Street address (Overture's freeform address). |
| `address_line_2` | Always empty — Overture has no second address line. The column exists to match the import format. |
| `city` | Locality. |
| `state_or_region` | State, province, or region. |
| `postal_code` | Postal / ZIP code. |
| `country_code` | ISO 3166-1 alpha-2 code, e.g. `US`. |
| `latitude` | Decimal degrees. |
| `longitude` | Decimal degrees. |

```csv
name,address_line_1,address_line_2,city,state_or_region,postal_code,country_code,latitude,longitude
Reading Terminal Market,1136 Arch St,,Philadelphia,PA,19107,US,39.9526,-75.1652
```

Overture's address coverage varies by region — some rows have a name and
coordinates but blank address columns. Size scales with the area requested;
a large city can produce a file in the tens of MB.

## Authentication

Access tokens are JWTs signed with **ES256**. The service verifies them against
the JSON Web Key Set at the `--jwks` URL, matching the token's `kid` header to
a key in the set. Tokens **must** carry an `exp` (expiration) claim; expired,
unsigned, or otherwise invalid tokens are rejected and logged.

The key set is fetched once at startup and refreshed automatically, including
when a token arrives with an unknown `kid` — so key rotation on the issuer side
needs no restart.

For local development, `--debug=true` disables all of this.
