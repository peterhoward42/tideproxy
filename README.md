# tideproxy

A small HTTP proxy in front of the [WorldTides](https://www.worldtides.info/) API. It exposes a single read-only endpoint that returns high and low tide extremes for a location over a fixed UTC window, with chart datum, no caching, and upstream attribution preserved.

The authoritative human-readable product and API description lives in [`docs/specs/overview.md`](docs/specs/overview.md). A machine-readable OpenAPI 3 description of the same HTTP surface (including CORS preflight and error codes implemented in code) is in [`openapi.yaml`](openapi.yaml) at the repository root.

## What it does

- **Endpoint:** `GET /v1/tides` with required query parameters `lat` and `lon` (valid ranges per the spec).
- **Semantics:** Extremes only, datum fixed to Chart Datum (`CD`), times in UTC. The response window runs from 00:00 UTC on the previous calendar day through 00:00 UTC three days after today (exclusive). Invalid input yields `400` with a structured error; upstream failures yield `502`; some server-side failures yield `500` (see [`openapi.yaml`](openapi.yaml)).
- **Deployment shape:** Google Cloud Function (2nd gen) via the Go [Functions Framework](https://github.com/GoogleCloudPlatform/functions-framework-go). Runtime configuration is environment variables (see [Setup and running](#setup-and-running)).

## Repository layout

- **`entry.go`** — Cloud Functions HTTP entry point ([`TidesProxy`](entry.go)); `make deploy` uses `--entry-point` `TidesProxy` by default.
- **`cmd/tideproxy/main.go`** — Local server: same handler, listens on `PORT` or `8080`.
- **`internal/app/`** — Request validation, upstream call, response mapping, CORS, tests.
- **`docs/specs/overview.md`** — Full API specification.
- **`openapi.yaml`** — OpenAPI 3 schema for `GET`/`OPTIONS` on `/v1/tides`.

## Setup and running

Operational commands live in the [`Makefile`](Makefile). Override deploy defaults there (`GCP_PROJECT_ID`, `GCP_REGION`, `CF_NAME`, alert-state bucket/path, etc.).

**Configuration.** Copy [`.env.example`](.env.example) to `.env` (gitignored), fill in secrets and GCS names, then export into your shell before `make` — **`make` does not load `.env`:**

```bash
set -a && source .env && set +a
```

### Production

Assumes a GCP project already exists and you can use `gcloud` against it. Default project id in the Makefile is `tides-proxy`.

| Step | Why | Command |
|------|-----|---------|
| Enable APIs | Cloud Functions (Gen2), build, logging, and GCS need project-level services | `make gcpsetup` |
| Alert dedupe bucket | Quota-exhaustion Telegram alerts use one GCS object for “last sent” UTC hour so cold starts and concurrent instances share state ([`docs/planning/telegram-quota-alert.md`](docs/planning/telegram-quota-alert.md)) | `make gcs-alert-state-setup` (once per project/bucket; idempotent) |
| Deploy | Publish code and env vars to the public HTTP function | `source .env` as above, then `make deploy` |
| Smoke test | Hit the deployed URL | `make examplerequestcommandcloud` |

Production deploy requires all variables in `.env.example`: WorldTides and Telegram credentials, plus `TELEGRAM_ALERT_STATE_BUCKET` and `TELEGRAM_ALERT_STATE_PATH` (both must be set in cloud; the function uses GCS only when both are non-empty).

**After code changes:** `source .env` if needed, `make deploy`, `make examplerequestcommandcloud`.

**After changing secrets or alert-state env vars:** update `.env`, `make deploy` (re-applies `--set-env-vars`).

**After changing bucket name or project:** adjust `.env` / Makefile defaults, run `make gcs-alert-state-setup` for the new bucket, then `make deploy`.

`make gcpsetup` again only if you add new GCP dependencies (new APIs).

### Local development

Same handler as production, via `cmd/tideproxy` and the Functions Framework. Differences from production:

| Production | Local |
|------------|--------|
| `make deploy` | `make runlocalproxysvr` (terminal 1) |
| Public function URL | `http://127.0.0.1:8080` — `make examplerequestcommandlocal` (terminal 2) |
| GCS alert dedupe required | **Optional:** leave `TELEGRAM_ALERT_STATE_BUCKET` and `TELEGRAM_ALERT_STATE_PATH` empty in `.env` — the app uses an in-memory noop store; quota alerts may fire on every exhaustion (useful while developing) |
| Function’s service account can read/write GCS | In cloud, `make gcs-alert-state-setup` grants the function’s runtime service account access to the alert-state bucket. Locally you skip that unless testing dedupe: set both GCS vars, then authenticate as yourself with `gcloud auth application-default login` (your user must be able to access that bucket) |

Still required locally: `WORLDTIDES_API_KEY`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` (see `.env.example`). `make runlocalproxysvr` does not require the GCS variables.

**Tests (no deploy, no `.env`):** `make gotest`

## Telegram notifications

The proxy sends optional Telegram messages on the same bot/chat as quota alerts (`TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`).

| Event | When | Message |
|-------|------|---------|
| Default app load | Successful `GET /v1/tides` whose validated `lat`/`lon` fall within the Looe bounding box (East Looe, West Looe, and Looe town) | `tideproxy:load:default` |
| Custom location load | Any other successful tide request | `tideproxy:load:custom` |
| Credits exhausted | WorldTides upstream error indicates quota exhaustion (deduped to once per UTC hour in production via GCS) | `tideproxy: WorldTides monthly API credits exhausted` |

Load notifications fire **once per successful response**, with no dedupe. Failed upstream calls are silent for load alerts.

**Configuration** (in [`internal/app/telegram_load_notify.go`](internal/app/telegram_load_notify.go)):

- `looeDefaultLoadLatMin` / `LatMax` / `LonMin` / `LonMax` — bounding box on validated coordinates for the client’s default Looe location fetch.
- `telegramLoadNotificationsEnabled` — compile-time kill switch for load notifications only. Set to `false` and redeploy to silence load pings during a traffic spike; credit-exhaustion alerts are unaffected.
