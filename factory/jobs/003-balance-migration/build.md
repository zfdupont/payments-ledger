## Round 1
Commits: 69a8ab4 Add ledger core migration with deferred balance trigger
Changed: migrations/0001_ledger_core.up.sql, migrations/0001_ledger_core.down.sql, internal/ledger/balance_test.go, go.mod, go.sum
Acceptance criteria:
- [x] AC1 tables, bigint amount_minor with CHECK > 0, char(3) currency, no float/numeric columns (file read; applied by TestBalance)
- [x] AC2 constraint triggers journal_entries_balanced and journal_lines_balanced, AFTER INSERT, DEFERRABLE INITIALLY DEFERRED, FOR EACH ROW, execute check_journal_entry_balanced() (file read)
- [x] AC3 unbalanced / one_sided / balanced_total_not_per_currency / no_lines: inserts succeed, COMMIT fails with 23514, exact messages asserted, entry count 0 (TestBalance/*)
- [x] AC4 balanced / balanced_multi_currency commit and row exists (TestBalance/balanced, TestBalance/balanced_multi_currency)
- [x] AC5 DATABASE_URL unset -> `--- SKIP: TestBalance`, exit 0; port 5999 -> skip with "Postgres unreachable at DATABASE_URL: ...", total 0.6 s (TestBalance)
- [x] AC6 after the AC3 run, `select count(*) from pg_namespace where nspname like 'ledger_test_%'` printed 0
- [x] AC7 down migration is comments only
- [x] AC8 go.mod requires github.com/jackc/pgx/v5 v5.13.0 as the only direct dependency; go.sum committed
Commands: build (go build ./...) -> pass; test (go test ./...) -> pass (DB test skips without DATABASE_URL; also passed 6/6 subtests with compose DATABASE_URL); lint (go vet ./...) -> pass
Notes:
- The balance check selects the first unbalanced currency using GROUP BY currency HAVING (debits <> credits) ORDER BY currency LIMIT 1, which is equivalent to the spec's "where sums differ, first currency" wording.
- go mod tidy also added indirect deps pgpassfile, pgservicefile and golang.org/x/text.
- Compose stack left running (was already up). No stray `ledger` binary in the worktree root. Row-level CHECKs are in the schema but, per spec, not covered by tests.
