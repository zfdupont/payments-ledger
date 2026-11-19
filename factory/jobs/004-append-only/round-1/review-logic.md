VERDICT: PASS

No defects found. Checked against the spec, not build.md.

- AC1/AC2: `migrations/0002_ledger_append_only.up.sql` defines `reject_ledger_mutation()` with `ERRCODE = 'restrict_violation'` and the exact message and hint. The three triggers are `BEFORE UPDATE OR DELETE OR TRUNCATE ... FOR EACH STATEMENT`. The down file is comment-only. `git diff main -- migrations/0001_ledger_core.up.sql` is empty.
- AC3: I re-ran `go test -count=1 -v ./internal/ledger/` against compose Postgres. All 11 `TestAppendOnly` subtests pass, including the zero-row UPDATE, `TRUNCATE ... CASCADE` and the `ON CONFLICT DO UPDATE` case. Each checks SQLSTATE 23001 and the exact message. The post-checks confirm counts and values are unchanged.
- AC4: `TestReversingEntry` passes. It checks 2 entries, 4 lines, the original lines unchanged, and net 0 for `cash` and `revenue`.
- AC5: `TestBalance` passes all six subtests with both migrations applied. `connect` uses `filepath.Glob` and `slices.Sort`, and it calls `t.Fatalf` when no files match or a file fails to apply.
- AC6: with `DATABASE_URL` unset, all three tests print `--- SKIP`.
- AC7: the `ledger_test_*` schema count is 0 after the run.
- Commands: `commands.md` is at HEAD 7a0961d, which matches the worktree HEAD. Build, test and vet all pass.
- Scope: only the four expected files changed. There is no non-test Go code, and `go.mod`, compose and `.env.example` are untouched.
- Logic: the statement-level triggers reject UPDATE and DELETE even when no rows match. No ledger row can be changed after commit. The balance trigger from 0001 still applies to reversing inserts. The function reads no tables, so `search_path` is not a concern for it.

Non-blocking notes (not defects):
1. `internal/ledger/append_only_test.go`: the subtest names embed the raw SQL, which makes them long. This is cosmetic.
2. The spec's out-of-scope bypasses (`DISABLE TRIGGER`, `session_replication_role`, owner `DROP TRIGGER`) are correctly left for the DB roles work.
