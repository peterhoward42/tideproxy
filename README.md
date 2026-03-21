# tideproxy

A small HTTP proxy in front of the [WorldTides](https://www.worldtides.info/) API. It exposes a single read-only endpoint that returns high and low tide extremes for a location over a fixed UTC window, with chart datum, no caching, and upstream attribution preserved.

The authoritative product and API description lives in [`docs/specs/overview.md`](docs/specs/overview.md). This README summarizes that intent and points to how you work on the code locally.

## What it does

- **Endpoint:** `GET /v1/tides` with required query parameters `lat` and `lon` (valid ranges per the spec).
- **Semantics:** Extremes only, datum fixed to Chart Datum (`CD`), times in UTC. The response window runs from 00:00 UTC today through 00:00 UTC three days later (exclusive). Invalid input yields `400` with a structured error; upstream failures yield `502`.
- **Deployment shape:** Intended as a Google Cloud Function (2nd gen) using the Go [Functions Framework](https://github.com/GoogleCloudPlatform/functions-framework-go) with source-based deployment. The API key for WorldTides is supplied via the `WORLDTIDES_API_KEY` environment variable at runtime (see the spec for the full contract).

## Repository layout

- **`main.go`** — Registers the HTTP handler and starts the framework (local default port `8080` unless `PORT` is set).
- **`internal/app/`** — Request validation, upstream call, response mapping, CORS wrapper, and tests.
- **`docs/specs/overview.md`** — Full API specification (JSON shapes, error codes, implementation notes).
- **`docs/prompts/`** — Incremental prompts used to drive implementation; much of this codebase was built by working through those steps with Cursor agent skills applied for Go style, testing, and related conventions.

## Development

Common workflows (tests, local run, deploy, example `curl`, and a one-shot “start server and hit it” helper) are defined as **`make` targets** in the [`Makefile`](Makefile). Read that file for exact commands, variables (`GCP_REGION`, `CF_NAME`, etc.), and prerequisites such as `WORLDTIDES_API_KEY` for targets that need it.

The default goal in the `Makefile` is `gotest`; other targets cover local server, deploy, and example requests—see the file for names and usage.
