VERDICT: PASS

Reviewed d0923cc (HEAD matches commands.md). Build, vet and test (DB set, DB unset, race) all pass per commands.md. AC9 grep prints nothing, and 0001/0002/cmd/go.mod/go.sum are unchanged (re-checked).

Contract checks:
- Idempotency-Key: missing or empty gives 400 `idempotency_key_required`, checked before the body is read or a tx begins (internal/api/idempotency.go:76-84). Over 255 bytes gives 400 `idempotency_key_invalid`. Replay returns the stored status and bytes via `writeRaw` (idempotency.go:100-108). A hash mismatch gives 422 `idempotency_key_reused` and writes nothing. Hash is SHA-256 over METHOD, path and raw body (idempotency.go:192-197), as specified.
- Race handling: the INSERT uses `ON CONFLICT DO NOTHING RETURNING 1`. On no row it rolls back and re-reads in a fresh tx (idempotency.go:125-137, 152-169), as in the spec. It also runs the handler's writes in the same tx as the record, so a rollback undoes both.
- Only 2xx is stored: handler errors and non-2xx statuses return before the INSERT (idempotency.go:110-123), and the table CHECK enforces 200-299.
- Error mapping (idempotency.go:44-63): `*apiError` as given, 23514 passes the Postgres message, other class 23 gets the generic message, everything else is 500 `internal` with no detail. COMMIT errors go through `writeError`, so the deferred balance trigger surfaces as 422 (journal_entries_test.go:79-107 covers it, including retry with the same key).
- Amounts are `int64` `amount_minor` plus `currency`. `DisallowUnknownFields` and the trailing-data check give `invalid_body` for fractional, unknown-field, bad JSON and trailing data (journal_entries.go:33-42). No floats.
- `addWrite` is the only registration path and always wraps with the idempotency layer (router.go:50-53). `New(nil)` does not panic.
- Tests: `writeCases` plus `coverage` guard every write route (idempotency_test.go:29-108). `TestWriteRoutesIdempotent` covers missing key, replay, reuse and replay-after-reuse over `httptest` with row counts. Journal tests cover 201 rows, 422 unbalanced, 23503 generic, 400 invalid body and the 256-byte key.
- No breaking changes to existing shapes. This is the first endpoint, and the spec says so.

Non-blocking notes (no change required for this round):
1. internal/api/idempotency.go:81. A key that is under 255 bytes but invalid UTF-8 (HTTP header obs-text is allowed through by net/http) will fail the `text` bind or SELECT in Postgres (SQLSTATE 22021, class 22) and return 500. Client input should be a 400 `idempotency_key_invalid`. Consider a `utf8.ValidString` check, and widen the message when this is added. The spec did not define it, so it is not a defect against the spec.
2. The concurrent same-key path (`replayCommitted`) has no test, which build.md acknowledges. The spec listed only sequential cases. A two-goroutine test with a `pgxpool` would pin it, and could go in the auth or outbox job.
3. Mux-generated 404 and 405 responses are `text/plain`, not the JSON error shape. No existing endpoint sets a precedent, and nothing is served yet. Worth deciding before the router is served from a binary.
