## Round 1
Commits: d0923cc Add idempotent POST /journal-entries, router, and write-route coverage test (migration 0003)
Changed: migrations/0003_idempotency_keys.up.sql, migrations/0003_idempotency_keys.down.sql, internal/ledger/ledger.go, internal/api/router.go, internal/api/idempotency.go, internal/api/journal_entries.go, internal/api/db_test.go, internal/api/idempotency_test.go, internal/api/journal_entries_test.go
Acceptance criteria:
- [x] AC1 write route coverage without DB (TestWriteRoutesCovered)
- [x] AC2 coverage helper (TestIdempotencyCoverageHelper)
- [x] AC3 missing key / replay / reused key against Postgres (TestWriteRoutesIdempotent/POST_/journal-entries)
- [x] AC4 201 `{"id":N}` and exactly the two posted lines (TestPostJournalEntryCreatesRows)
- [x] AC5 unbalanced 422 with trigger message, no rows, same key then succeeds (TestPostJournalEntryUnbalanced; also TestPostJournalEntryUnknownAccount for the 23503 generic message)
- [x] AC6 fractional amount, unknown field, 256-byte key (TestPostJournalEntryInvalidBody, TestPostJournalEntryKeyTooLong; bad JSON and trailing data also covered)
- [x] AC7 with DATABASE_URL unset, `go test -count=1 -v ./internal/api/ ./internal/ledger/` exits 0; all DB tests SKIP, the two no-DB tests PASS (ran it)
- [x] AC8 migration 0003 matches the Design; `git diff main --stat` on 0001/0002 up, cmd/, go.mod, go.sum is empty
- [x] AC9 the grep prints nothing (exit 1)
- [x] AC10 after the DB run, the `ledger_test_%` namespace count printed 0
Commands: build (go build ./...) -> pass; test (go test ./..., DATABASE_URL set, and the api/ledger packages also run with it unset) -> pass; lint (go vet ./...) -> pass
Notes:
- Compose stack was already up and was left running. No stray `ledger` binary in the worktree root.
- `writeCase.body` is a func (like `otherBody`) because the body needs the seeded account ids.
- On a concurrent same-key conflict (ON CONFLICT returns no row), the code rolls back and re-reads in a fresh transaction via `Router.replayCommitted`. This path is not exercised by a test (spec only listed the sequential cases).
- Only the idempotency row's non-2xx handler results are returned as-is (marshaled resp, status) and rolled back; the one route today never returns a non-2xx status without an error.
- The commit carries no attribution trailer, per the user's global CLAUDE.md.
