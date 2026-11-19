VERDICT: PASS

Reviewed d0923cc (HEAD of factory/005-idempotency-test) against main. commands.md records the same sha, and build, test (DB set and unset), vet and the race run are all green. I did not rerun them. No defects found.

Checklist:

1. Balance: PASS. `internal/ledger/ledger.go:23-46` does no Go-side balance check. It only inserts. The deferred constraint triggers from `0001` (`journal_entries_balanced`, `journal_lines_balanced`) fire at the idempotency layer's `tx.Commit` (`internal/api/idempotency.go:~148`). A commit error goes through `writeError`. SQLSTATE 23514 returns 422 with the trigger message, and the idempotency insert rolls back with it. The idempotency row is inserted before COMMIT in the same transaction, so a failed balance check leaves the key unused.

2. Append-only: PASS. The diff has no UPDATE or DELETE on ledger tables. `ledger.go` only does INSERT. `idempotency_keys` is never updated. It is inserted once, already complete. It is not a ledger table, and the `0003` header says so.

3. Idempotency: PASS.
   - `Router.addWrite` (`router.go:~50`) is the only registration path for write routes. It always wraps the handler with `r.idempotent`.
   - A missing or empty key gets 400 before the body is read or a transaction begins.
   - A key over 255 bytes gets 400.
   - The SELECT is followed by an `INSERT ... ON CONFLICT (client_id, idempotency_key) DO NOTHING RETURNING 1`, backed by the primary key. The concurrent-writer case (ErrNoRows) rolls back and re-reads in a fresh transaction. Under READ COMMITTED the second writer waits on the unique index, then sees the committed row. A different hash returns 422 `idempotency_key_reused`.
   - A replay returns the stored bytes verbatim. The `bytea` column avoids jsonb re-serialization.
   - Non-2xx results and errors roll back and store nothing.
   - `TestWriteRoutesCovered` and `TestWriteRoutesIdempotent` guard future write routes.
   - The concurrent-conflict path (`replayCommitted`) has no test. build.md notes this, and the spec did not require it. It is not a defect.

4. Outbox and events: PASS. `grep -rniE 'INSERT INTO (journal_|accounts)|kafka|outbox|float' internal/api` over non-test files prints nothing. The handler publishes nothing, and the outbox is explicitly out of scope per spec. `cmd/` is unchanged, so the router is not served. Unauthenticated money movement stays unreachable.

5. Money: PASS. `Line.AmountMinor` is `int64` and `journalLineRequest.AmountMinor` is `int64`. Fractional JSON numbers fail decode and return 400 `invalid_body`. Currency is a three-letter code, enforced by the `0001` CHECK `^[A-Z]{3}$`. Amounts must be above zero, also from `0001`. There are no floats in the path.

6. Migrations: PASS. `0003_idempotency_keys.{up,down}.sql` are new files. The up file matches the spec's Design: primary key, key length CHECK, hash length CHECK, status 2xx CHECK. It has no BEGIN/COMMIT and no schema qualification. `git diff main --stat` on `0001`, `0002`, `cmd/`, `go.mod` and `go.sum` is empty. The diff adds only 9 new files.

Non-blocking observations, no action required:
- `writeError` returns the raw Postgres message to the client for 23514, as the spec specifies. A row CHECK violation, such as a lowercase currency, therefore exposes the constraint and relation names. The spec accepts this.
- `idempotency_keys` has no outbox row yet. The outbox job must add it inside the same transaction, after the `PostEntry` call and before the idempotency insert or COMMIT.
