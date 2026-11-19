VERDICT: PASS

No defects found. Checked at HEAD 69a8ab4, which matches the sha in commands.md. All three commands pass there: build, test (with and without DATABASE_URL), and vet.

Checklist:
1. Ledger writes only in the ledger service package: pass. The diff has no handlers or service code. The only writes to ledger tables are in `internal/ledger/balance_test.go`, a test-only file in the ledger package. The migration is the only other place that touches them.
2. Naming: pass. Tables and columns are snake_case (`accounts`, `journal_entries`, `journal_lines`, `entry_id`, `account_id`). Money is `amount_minor bigint` plus `currency char(3)`. There are no floats or `numeric` columns. The `numeric` variables inside the function are only for overflow-safe sums.
3. Pattern `.mex/patterns/add-migration.md`: pass.
   - Up and down files use the `{version}_{title}` naming.
   - The down file is a documented no-op, as step 2 allows.
   - Balance is enforced by a deferred constraint trigger, as the Gotchas say.
   - The migration is new and edits no applied migration.
   - The one deviation from the spec is `GROUP BY ... HAVING ... ORDER BY currency LIMIT 1` instead of a `WHERE`. It is documented in build.md, is equivalent, and returns the alphabetically first unbalanced currency.
4. Dead code and speculative abstraction: none. `accountFor` and `line` are small test helpers that are used. The only new direct dependency is pgx, as the spec asks. The indirect deps in `go.mod` come from `go mod tidy`.
5. Env vars: no new ones. `DATABASE_URL` is already listed in `.env.example` and in `CLAUDE.md`.
6. `.mex/` scaffold: not updated in this diff, and the spec puts that out of scope ("Updating `.mex/`, `CLAUDE.md` or the README. The post-job mex update handles those."). Per CLAUDE.md Workflow, the post-job update should cover these items:
   - `.mex/context/stack.md:57` still has "Postgres driver TBD". It should now say pgx v5.
   - `.mex/context/conventions.md:47` still has `[VERIFY AFTER FIRST IMPLEMENTATION]` on snake_case, and the Layout line still says `migrations/` is planned.
   - `.mex/patterns/add-migration.md:34` still has its `[VERIFY ...]` tag.

   None of this is a defect in the round.
