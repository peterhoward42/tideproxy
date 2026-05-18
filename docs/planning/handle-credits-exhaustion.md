# Plan: surface WorldTides credit exhaustion to proxy clients

Working document for a multi-session change. Annotate **Open questions** inline when you have answers.

## Goal

When the proxy’s WorldTides API key has no credits left, `GET /v1/tides` should tell **proxy clients** that the failure is due to upstream quota exhaustion—not a generic upstream failure indistinguishable from network errors or bad JSON.

The proxy uses a single operator key (`WORLDTIDES_API_KEY`); credit exhaustion is an infrastructure/operator condition, not invalid input from the caller.

## Conclusion

**Feasible.** WorldTides v3 returns structured failures in JSON (`status`, `error`). The proxy already receives those bodies but does not parse or map them. A dedicated proxy error code and HTTP status are a spec + handler change, not a new upstream integration.

## Rationale (upstream)

From [WorldTides apidocs](https://www.worldtides.info/apidocs):

- `status` — documented as the HTTP status of the response; `200` is success.
- `error` — when `status` ≠ 200, a description of the failure. Docs encourage treating these strings as stable identifiers; new strings may be added later.

Verified with a live probe (invalid key): HTTP `400` and body `{"status":400,"error":"API key is invalid"}` — HTTP status and JSON `status` align.

Credit exhaustion is not listed in public docs, but the same envelope is the intended detection path (likely a stable `error` string such as “Not enough credits”; **must be confirmed empirically** before hard-coding).

Operational note: proxy requests ~4 UTC calendar days of extremes (`start` + `length` in `SynthesiseOutputRequest`). WorldTides bills extremes at 1 credit per 7 days, so each successful call is likely **1 credit**.

## Rationale (current proxy gap)

Today in `handleTides` (`internal/app/tides.go`):

1. Non-2xx HTTP → `502` + `UPSTREAM_ERROR` without reading the body.
2. HTTP 2xx but body fails `ParseIncomingResponse` (e.g. JSON `status` ≠ 200) → same generic `502`.

`ParseIncomingResponse` (`internal/app/incoming_response.go`) only validates success payloads; it does not read upstream `error`.

All upstream failures (credits, bad operator key, no location, malformed body) collapse to the same client shape defined in `docs/specs/overview.md` and `openapi.yaml`.

## Proposed direction (implementation sketch)

Not for this doc’s scope to implement—order of work TBD across sessions:

1. **Confirm** real credit-exhaustion response: HTTP status, JSON `status`, exact `error` string (depleted account or WorldTides support).
2. **Spec / OpenAPI** — add error case (code name, HTTP status, message policy) to `docs/specs/overview.md` and `openapi.yaml`.
3. **Upstream error parse** — lightweight decode of `{status, error}` on failed upstream calls (read body even when HTTP ≠ 2xx if JSON is present).
4. **Handler mapping** — e.g. known credit `error` → dedicated code (e.g. `UPSTREAM_CREDITS_EXHAUSTED`) and **503** (service unavailable for callers); keep `UPSTREAM_ERROR` / **502** for other upstream failures; operator misconfiguration (e.g. invalid API key) → **500** / `INTERNAL_ERROR`.
5. **Tests** — table-driven cases from captured or fixture upstream error bodies; keep `ParseIncomingResponse` focused on success only.

## Open questions

> Add answers below each item (or inline). Remove or resolve items as we go.

### 1. Exact upstream signal

What HTTP status and exact `error` string does WorldTides return when credits are exhausted?

- *Your answer: rather than the tricky challenge of researching this, I suggest that any error response that includes the substring "credit" will be good enough.*

### 2. Proxy error code and HTTP status

Preferred client-facing `error.code` and HTTP status?

- Default proposal: `UPSTREAM_CREDITS_EXHAUSTED` + **503**.
- Alternatives: **502** with distinct code; **429**; **402** (usually reserved for “caller must pay”—less fitting since clients do not hold the WorldTides key).
- *Your answer: your default*

### 3. Client-visible message

Fixed message (e.g. “Tidal data temporarily unavailable”) vs forwarding upstream `error` text vs a hybrid?

- *Your answer: "Monthly API credits exhausted"*

### 4. Other upstream `error` strings

Should we explicitly map any other WorldTides errors in this change (e.g. `No location found`), or only credits in v1?

- *Your answer: no*

### 5. Invalid / misconfigured operator API key

Treat upstream `API key is invalid` as `INTERNAL_ERROR` (**500**), not `UPSTREAM_ERROR`—agree?

- *Your answer: unchanged*

### 6. Empirical verification

How will we capture the real credit-exhaustion response (test account, staging key, manual curl log, support ticket)?

- *Your answer: We likely won't bother*

### 7. Rollout / compatibility

Any consumers that depend on today’s generic `502` / `UPSTREAM_ERROR` only? Breaking-change tolerance?

- *Your answer: not concerned - I own the only client and it is in development and I am the only user*

## References

- Product spec: `docs/specs/overview.md`
- OpenAPI: `openapi.yaml`
- Handler: `internal/app/tides.go`
- Success parse: `internal/app/incoming_response.go`

