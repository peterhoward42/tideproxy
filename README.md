# tideproxy

A small HTTP proxy in front of the [WorldTides](https://www.worldtides.info/) API. It exposes a single read-only endpoint that returns high and low tide extremes for a location over a fixed UTC window, with chart datum, no caching, and upstream attribution preserved.

The authoritative human-readable product and API description lives in [`docs/specs/overview.md`](docs/specs/overview.md). A machine-readable OpenAPI 3 description of the same HTTP surface (including CORS preflight and error codes implemented in code) is in [`openapi.yaml`](openapi.yaml) at the repository root.

## What it does

- **Endpoint:** `GET /v1/tides` with required query parameters `lat` and `lon` (valid ranges per the spec).
- **Semantics:** Extremes only, datum fixed to Chart Datum (`CD`), times in UTC. The response window runs from 00:00 UTC on the previous calendar day through 00:00 UTC three days after today (exclusive). Invalid input yields `400` with a structured error; upstream failures yield `502`; some server-side failures yield `500` (see [`openapi.yaml`](openapi.yaml)).
- **Deployment shape:** Google Cloud Function (2nd gen) via the Go [Functions Framework](https://github.com/GoogleCloudPlatform/functions-framework-go). Runtime secrets are environment variables (see [Development](#development)).

## Repository layout

- **`entry.go`** — Cloud Functions HTTP entry point ([`TidesProxy`](entry.go)); gcloud `--entry-point` must match.
- **`cmd/tideproxy/main.go`** — Local server: same handler, listens on `PORT` or `8080`.
- **`internal/app/`** — Request validation, upstream call, response mapping, CORS, tests.
- **`docs/specs/overview.md`** — Full API specification.
- **`openapi.yaml`** — OpenAPI 3 schema for `GET`/`OPTIONS` on `/v1/tides`.

## Development

Workflows are **`make` targets** in the [`Makefile`](Makefile) (`gotest`, `runlocalproxysvr`, `deploy`, example `curl`s). Override deploy defaults there (`GCP_REGION`, `CF_NAME`, etc.).

### Secrets and local run

Copy [`.env.example`](.env.example) to `.env` at the repo root (`.env` is gitignored), fill in values, then export them — **`make` does not load `.env` itself**:

```bash
set -a && source .env && set +a
```

**Terminal 1** — start the local server (requires the three variables above):

```bash
make runlocalproxysvr
```

**Terminal 2** — example request against `http://127.0.0.1:8080`:

```bash
make examplerequestcommandlocal
```

The same variables must be exported for `make deploy` (passed through to the function as Cloud env vars).

### Telegram quota alerts (GCS dedupe)

When WorldTides credits are exhausted, the function can send a Telegram alert. In production, set both:

- `TELEGRAM_ALERT_STATE_BUCKET` — GCS bucket (e.g. `tides-proxy-telegram-alert-state`, globally unique)
- `TELEGRAM_ALERT_STATE_PATH` — object path in that bucket (e.g. `telegram-quota-alert/last-sent-hour.txt`)

The object stores the UTC calendar hour (`2006-01-02T15`) of the last **successful** alert so at most one message is sent per hour. Omit both variables locally to skip GCS (alerts still fire on every exhaustion, as in iteration 2).

Provision once per project (replace bucket name if taken):

```bash
export GCP_PROJECT_ID=tides-proxy
export TELEGRAM_ALERT_STATE_BUCKET=tides-proxy-telegram-alert-state
export TELEGRAM_ALERT_STATE_PATH=telegram-quota-alert/last-sent-hour.txt

gcloud config set project "$GCP_PROJECT_ID"
gcloud storage buckets create "gs://${TELEGRAM_ALERT_STATE_BUCKET}" --location=europe-west1

# After deploy, grant the function's runtime service account object access on that path
# (default compute SA: PROJECT_NUMBER-compute@developer.gserviceaccount.com)
```

`make gcpsetup` enables the Storage API. `make deploy` passes the bucket and path env vars when exported.
