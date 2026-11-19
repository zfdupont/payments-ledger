## Round 1
Commits: 7a0961d Add append-only triggers on ledger tables (migration 0002)
Changed: migrations/0002_ledger_append_only.up.sql, migrations/0002_ledger_append_only.down.sql, internal/ledger/balance_test.go (connect helper only), internal/ledger/append_only_test.go
Acceptance criteria:
- [x] AC1 function + three statement-level triggers (TestAppendOnly via connect applying 0002)
- [x] AC2 down is comment-only; `git diff main -- migrations/0001_ledger_core.up.sql` is empty
- [x] AC3 all TestAppendOnly subtests pass with 23001 and exact messages, including zero-row UPDATE and the ON CONFLICT DO UPDATE case; counts/values unchanged afterwards (TestAppendOnly)
- [x] AC4 reversal commits, 2 entries / 4 lines, originals unchanged, net 0 for cash and revenue (TestReversingEntry)
- [x] AC5 TestBalance passes all six subtests with both migrations applied (TestBalance)
- [x] AC6 DATABASE_URL unset: exit 0, SKIP for TestBalance, TestAppendOnly, TestReversingEntry; unreachable URL gives the "Postgres unreachable" message
- [x] AC7 ledger_test_* schema count is 0 after the run
Commands: build -> pass (go build ./...; stray `ledger` binary removed), test -> pass (with and without DATABASE_URL), lint -> pass (go vet ./...)
Notes: The ON CONFLICT DO UPDATE case is an extra subtest beyond the spec's list (spec Behavior covers it). Subtest names embed the SQL text. Compose stack left running as instructed; I did not start or stop it.
