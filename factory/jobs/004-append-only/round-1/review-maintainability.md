VERDICT: PASS

Reviewed HEAD 7a0961d against main. It matches the commands.md sha, and build, test (with and without DATABASE_URL) and vet all passed there.

No defects. Checklist results:

1. Ledger table writes: the diff adds no non-test Go code. The only writes to ledger tables are in test files (`internal/ledger/append_only_test.go`, `postNamed`), and no handler code exists. Pass.
2. Naming: the triggers (`accounts_append_only`, `journal_entries_append_only`, `journal_lines_append_only`), the function `reject_ledger_mutation` and the migration files `0002_ledger_append_only.{up,down}.sql` are snake_case and follow the `NNNN_<name>` layout. The tests use `amount_minor` plus `currency`. Pass.
3. Pattern `.mex/patterns/add-migration.md`: 0002 is a new migration and 0001 is untouched (the diff shows no 0001 change). The down migration is a documented comment-only no-op, as step 2 allows. The invariant is enforced in the database (step 3). The up file has no BEGIN/COMMIT and no schema qualification, like 0001, and its header explains why CASCADE is needed. Pass.
4. Dead code and speculation: nothing unused. `os`, `filepath` and `slices` are all used in `balance_test.go`. The `connect` helper change is limited to applying every `*.up.sql` in sorted order, as the spec says. The extra `INSERT ... ON CONFLICT DO UPDATE` subtest is covered by spec Behavior, so it is not speculative. Pass.
5. Env vars: none added, so `.env.example` and the CLAUDE.md Environment Variables section need no change. Pass.
6. `.mex/` scaffold: the spec puts `.mex/` updates in the post-job mex update, and CLAUDE.md Workflow requires that step after the job. Nothing in this diff is a defect. At that step, update the conventions Layout line and the `ROUTER.md` project state to note that append-only is now enforced by triggers (add-migration.md Gotchas still reads "revoke UPDATE/DELETE or add a trigger"). Pass.

Non-blocking note: subtest names embed the full SQL text (`tc.table+"/"+tc.op+"/"+tc.sql`). This is long but unambiguous, since the same table and op pair appears more than once. No change needed.
