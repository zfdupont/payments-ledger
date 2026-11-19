VERDICT: PASS

No blocking defects. I ran nothing from commands.md again (HEAD d0923cc matches). I did probe concurrency myself and re-checked AC8 and AC9.

Checks against the review list:

- Concurrent same-key requests. `internal/api/idempotency.go:117-131` inserts with `ON CONFLICT (client_id, idempotency_key) DO NOTHING RETURNING 1`. The primary key in `migrations/0003_idempotency_keys.up.sql` is the arbiter, not the earlier SELECT. The loser blocks on the winner's index entry. It then rolls back and replays the committed row through `replayCommitted`. Probe: I copied the worktree to /tmp, added a throwaway test, and ran 20 rounds of 10 parallel requests with one key. Each request used its own connection, and 9 sent the same body and 1 sent a different one. Every round wrote exactly one entry and 2 lines. The 9 same-body requests returned the identical 201 `{"id":N}`. The different-body request returned 422 `idempotency_key_reused`. The test passed under `-race`. The temp copy is deleted, no `ledger_test_%` schemas remain, and the worktree is clean.
- Conflicting replay. `lookup` compares the stored hash and returns `errKeyReused` (422) on a mismatch. This also holds on the concurrent path, where `replayCommitted` calls the same function. The sequential test covers it, and a further replay with the original body still returns the original response.
- Rollback and definite status. Every error and non-2xx path returns before `tx.Commit`, and `defer tx.Rollback` runs. The deferred balance trigger surfaces at COMMIT and is mapped by `writeError` (23514 gives the trigger message, other class 23 gives the generic 422, anything else gives 500 with no detail). Nothing is stored for a failed request, so a retry with the same key and a corrected body works. `TestPostJournalEntryUnbalanced` and `TestPostJournalEntryUnknownAccount` cover this.
- Spec points. The key check runs before the body read and `Begin`. The hash is SHA-256 over method, path and raw bytes. Replays write the stored bytes. Decoding rejects unknown fields, trailing data and fractional amounts. `PostEntry` does no Go-side balance check. Zero lines or `null` are rejected by the 0001 "no lines" trigger at COMMIT.
- AC8 and AC9. `git diff main --stat` on `cmd/`, `go.mod`, `go.sum` and the 0001/0002 up migrations is empty. The grep finds no INSERT into the ledger tables, no Kafka and no outbox code in non-test `internal/api` code.
- Temporal, outbox relay, ingestion and rate limit. None of these are in this job's scope, and there is no code for them in the diff.

Non-blocking notes (no change required this round):

1. `internal/api/idempotency_test.go`: the concurrent same-key path (`replayCommitted`, `idempotency.go:122-125`) has no committed test. The spec did not require one, and my probe shows the path works. It needs a pool or a connection per request, and tests currently use one `*pgx.Conn`. Worth adding when a pool-based test helper exists, for example in the outbox job.
2. `internal/api/idempotency.go:117`: the non-2xx branch returns the marshaled `resp` directly. This matches the spec, but nothing exercises it, because the one route never returns a non-2xx status without an error. Add a test when a route needs it.
