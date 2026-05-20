# Planning: Telegram alert on WorldTides quota exhaustion

This proxy calls the WorldTides API. The proxy can already detect the operator condition where API usage credits for the month are exhausted (see `docs/planning/handle-credits-exhaustion.md` and `CreditsExhausted()` / upstream `error` containing `credit`, case-insensitive).

This document plans an **additional** reaction (not a replacement for client-facing behaviour): send a Telegram message to a personal account using an existing Telegram bot.

Each **iteration** below is intended to be implementable as a separate change / request to the codebase.

## Conclusions (alignment)

- **Audience and risk:** Single owner-operator client; no real users yet. Prefer **simplicity and coherence** over defensive gating (e.g. iteration 1 need not be feature-flagged off in production).
- **Latency:** On the exhaustion path the client is already unfit for purpose; **extra latency** (Telegram round-trip, GCS read/write) on that path is **acceptable** and the client already tolerates it.
- **Telegram failures:** If Telegram is down or uncooperative: **log to stdout**, **carry on** with the normal response path. **No retry loop** for sends.
- **Dedupe goal:** At most **one alert per UTC calendar hour** for the same exhaustion signal. Corner cases (cold start, multi-instance races) are intentionally deprioritised unless we tighten later.
- **Persistence:** Cloud Functions do not offer durable in-memory state across instances or recycling. Use **GCS** (single small object, e.g. JSON or plain text holding the last-sent UTC calendar hour) so dedupe survives cold starts and scales across instances. Configure bucket + IAM for the function’s service account (`storage.objects.get` and create/update on that object path as appropriate).
- **State update rule:** Update stored “last sent hour” only after a **successful** `sendMessage` (or equivalent), so a failed send does not suppress a later attempt in the same hour.
- **Telegram API:** It does not expose a queryable “when did I last send?” history for dedupe; **our** GCS object is the source of truth for last-sent hour.

## Incremental development

### Iteration 1

Create the Telegram **send** capability (bot token and chat identifier from environment; never commit secrets). **Exercise it on every incoming request** for immediate feedback while wiring is verified. No extra gating required given current deployment assumptions.

### Iteration 2

Emit the message **only** when handling the WorldTides **quota exhaustion** path — the same predicate used for credit exhaustion elsewhere (not on every request).

### Iteration 3 (implemented)

**Persisted dedupe via GCS:**

- Provision or designate a bucket (and object path) for this single piece of state; grant the function service account minimal object access on that path.
- Runtime configuration (deploy env vars, same pattern as Telegram secrets): `TELEGRAM_ALERT_STATE_BUCKET` (bucket name) and `TELEGRAM_ALERT_STATE_PATH` (object path within the bucket, e.g. `telegram-quota-alert/last-sent-hour.txt`). Payload: plain text UTC calendar hour `2006-01-02T15`.
- On exhaustion: read the object at that path (treat missing object as “never sent”). If the current **UTC calendar hour** matches the stored last-sent hour, skip Telegram. Otherwise send; on **success**, write the new hour back to the object.
- Optional later tightening: GCS generation preconditions to reduce same-hour double-send under concurrent invocations — not required for the initial simple story.

## Open operational notes

- Use **UTC** for the stored calendar hour bucket to avoid DST ambiguity.
- Keep the planning doc in sync if the exhaustion detection string or module layout changes in `handle-credits-exhaustion` work.
