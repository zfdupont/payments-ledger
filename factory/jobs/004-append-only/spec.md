# Make ledger tables append-only in Postgres

## Goal
Add migration `0002`. It puts statement-level `BEFORE UPDATE OR DELETE OR TRUNCATE` triggers on every ledger table that exists today: `accounts`, `journal_entries` and `journal_lines`. Each trigger raises an exception, so once a row is committed Postgres refuses to change or remove it. Add `internal/ledger/append_only_test.go`. It proves that UPDATE, DELETE and TRUNCATE fail on each table, and that a reversing entry (an INSERT) still commits.

This job puts the hard rule "ledger rows are append-only; corrections are reversing entries" into the database and weakens none. It also closes the balance bypass through UPDATE or DELETE of `journal_lines` that job 003's reviews pointed out.

`accounts` is covered because the architecture lists accounts as ledger data written only by the ledger service. A later feature that needs a mutable account attribute (for example a status) must relax this with its own migration and its own decision. As in job 003, the DB tests skip when `DATABASE_URL` is unset or unreachable, so `go test ./...` still passes without Postgres.

## Behavior
- The compose Postgres is up and `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable`. With both migrations applied, every statement below fails right away, at the statement and not at commit. Each failure has SQLSTATE `23001` (`restrict_violation`), message `<table> is append-only: <OP> is not allowed` and hint `Ledger rows are never changed; post a reversing journal entry to correct one.`
  - `UPDATE` and `DELETE` on `accounts`, `journal_entries` and `journal_lines`, whatever their `WHERE` clause. The trigger is statement-level, so an `UPDATE ... WHERE false` that matches no rows is rejected too.
  - `TRUNCATE journal_lines`, and `TRUNCATE journal_entries CASCADE` / `TRUNCATE accounts CASCADE`. Each message names the table given in the statement, for example `journal_entries is append-only: TRUNCATE is not allowed`. A plain `TRUNCATE journal_entries` or `TRUNCATE accounts` without `CASCADE` is already stopped by Postgres's FK check (`0A000`) before any trigger fires, so the test uses `CASCADE` to reach the trigger.
  - `INSERT ... ON CONFLICT DO UPDATE` on a ledger table is rejected with `<table> is append-only: UPDATE is not allowed`. `ON CONFLICT DO NOTHING` and plain `INSERT` are not affected.
- After any of these failures, no row has changed: the counts and column values are the same as before.
- Inserting a reversing entry commits. For example, if the original entry is debit `cash` 1000 USD / credit `revenue` 1000 USD, the reversal is a new `journal_entries` row with description `reverses entry <id>` and the lines credit `cash` 1000 USD / debit `revenue` 1000 USD. Afterwards both entries and all four lines exist, the original lines are unchanged, and each account's net balance (debits minus credits) is 0. The balance trigger from `0001` still applies to the new entry.
- `DROP SCHEMA ... CASCADE` (the test cleanup) is not blocked, so the test still leaves no `ledger_test_*` schema behind.
- With `DATABASE_URL` unset or unreachable, `TestAppendOnly` and `TestReversingEntry` skip with the same messages as `TestBalance` (`DATABASE_URL not set; skipping Postgres test` / `Postgres unreachable at DATABASE_URL: <error>`).

## Design
The SQL below ran against the compose Postgres 18.7 in a throwaway schema with `0001` applied, and every result listed under Behavior was observed. Keep this shape.

- `migrations/0002_ledger_append_only.up.sql` (new). Plain SQL with no `BEGIN`/`COMMIT` and no schema qualification, like `0001`. It has a header comment saying what it enforces, that corrections are reversing entries, and why `CASCADE` is needed to reach the TRUNCATE trigger on FK-referenced tables.
  ```sql
  CREATE FUNCTION reject_ledger_mutation() RETURNS trigger
  LANGUAGE plpgsql AS $$
  BEGIN
      RAISE EXCEPTION '% is append-only: % is not allowed', TG_TABLE_NAME, TG_OP
          USING ERRCODE = 'restrict_violation',
                HINT = 'Ledger rows are never changed; post a reversing journal entry to correct one.';
  END;
  $$;

  CREATE TRIGGER accounts_append_only
      BEFORE UPDATE OR DELETE OR TRUNCATE ON accounts
      FOR EACH STATEMENT EXECUTE FUNCTION reject_ledger_mutation();
  -- same for journal_entries_append_only and journal_lines_append_only
  ```
  The triggers are `FOR EACH STATEMENT` on purpose. TRUNCATE only supports statement-level triggers, and a statement trigger also rejects updates and deletes that match zero rows. The function reads no tables, so the open `search_path` follow-up from job 003 does not apply to it.
- `migrations/0002_ledger_append_only.down.sql` (new). A comment-only, documented no-op. Dropping these triggers would remove a hard rule, so the down migration does not do it (`.mex/patterns/add-migration.md` step 2). Do not edit `0001_*` (step 1).
- `internal/ledger/balance_test.go` (edit, helper only). `connect(t)` currently applies only `0001`. Change it to apply every `../../migrations/*.up.sql`: get them with `filepath.Glob`, sort them with `slices.Sort` so the order is explicit, and `conn.Exec` each file in turn. Call `t.Fatalf` if no files match or any file fails to apply. Leave `TestBalance`, `post`, `line` and `accountFor` as they are. `TestBalance` must still pass with `0002` applied, because it only inserts.
- `internal/ledger/append_only_test.go` (new), `package ledger`:
  - A small local helper posts an entry whose lines name their account explicitly: description, then `(account name, direction, amount_minor, currency)` per line, in one tx, returning the id and the commit error. `post`/`accountFor` cannot express a reversal, because they always debit `cash` and credit `revenue`.
  - `TestAppendOnly`:
    - It calls `connect(t)` and posts one balanced entry (debit `cash` 1000 USD, credit `revenue` 1000 USD).
    - It then runs a table-driven set of subtests, one per `(table, statement)`. Each runs in autocommit with `conn.Exec` and asserts `errors.As(err, &*pgconn.PgError)`, `Code == "23001"` and `Message == "<table> is append-only: <OP> is not allowed"`. The statements are:
      - `accounts`: `UPDATE accounts SET name = name || '_x' WHERE name = 'cash'`, `DELETE FROM accounts WHERE name = 'cash'`, `TRUNCATE accounts CASCADE`.
      - `journal_entries`: `UPDATE journal_entries SET description = 'edited' WHERE id = $1`, `DELETE FROM journal_entries WHERE id = $1`, `TRUNCATE journal_entries CASCADE`.
      - `journal_lines`: `UPDATE journal_lines SET amount_minor = 1 WHERE entry_id = $1`, `UPDATE journal_lines SET amount_minor = 1 WHERE false` (zero rows), `DELETE FROM journal_lines WHERE entry_id = $1`, `TRUNCATE journal_lines`.
    - After the subtests, it asserts that nothing changed: 2 `accounts` rows still named `cash` and `revenue`, 1 `journal_entries` row with description `''`, and 2 `journal_lines` rows for the entry, each with `amount_minor = 1000`.
  - `TestReversingEntry`:
    - It calls `connect(t)`, posts the original entry, then posts the reversal (description `reverses entry <id>`, credit `cash` 1000 USD, debit `revenue` 1000 USD).
    - It asserts a nil commit error, 2 entries and 4 lines, and that the original entry's lines are unchanged.
    - It asserts that `SELECT a.name, sum(CASE l.direction WHEN 'debit' THEN l.amount_minor ELSE -l.amount_minor END) FROM journal_lines l JOIN accounts a ON a.id = l.account_id GROUP BY a.name` returns 0 for both `cash` and `revenue`.
- No changes to `go.mod`, `docker-compose.yml`, `.env.example`, `factory/factory.yaml` or `cmd/`. No non-test Go code. If the builder starts compose Postgres, it runs `docker compose down` afterwards, without `-v`.

## Acceptance criteria
- [ ] AC1: `migrations/0002_ledger_append_only.up.sql` defines `reject_ledger_mutation()`, which raises with `ERRCODE = 'restrict_violation'`. It also defines the triggers `accounts_append_only`, `journal_entries_append_only` and `journal_lines_append_only`, each `BEFORE UPDATE OR DELETE OR TRUNCATE ... FOR EACH STATEMENT` (test: read the file; applied by `internal/ledger/balance_test.go` `connect`)
- [ ] AC2: `migrations/0002_ledger_append_only.down.sql` exists and contains only comments, and `migrations/0001_ledger_core.up.sql` is unchanged from `main` (test: read the files / `git diff main -- migrations/0001_ledger_core.up.sql` is empty)
- [ ] AC3: after `docker compose up -d --wait postgres`, `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable go test -count=1 -v ./internal/ledger/` passes every `TestAppendOnly` subtest. Each one sees UPDATE, DELETE or TRUNCATE on `accounts`, `journal_entries` or `journal_lines` fail with SQLSTATE `23001` and message `<table> is append-only: <OP> is not allowed`, including the zero-row `UPDATE ... WHERE false`. The test then confirms that row counts and values are unchanged (test: `internal/ledger/append_only_test.go`)
- [ ] AC4: in the same run, `TestReversingEntry` passes. The reversing entry commits, 2 entries and 4 lines exist, the original lines are unchanged, and the net balance of `cash` and of `revenue` is 0 (test: `internal/ledger/append_only_test.go`)
- [ ] AC5: in the same run, `TestBalance` still passes all six subtests with both migrations applied by `connect`, which loads every `migrations/*.up.sql` in sorted order (test: `internal/ledger/balance_test.go`)
- [ ] AC6: with `DATABASE_URL` unset, `go test -count=1 -v ./internal/ledger/` exits 0 and prints `--- SKIP` for `TestBalance`, `TestAppendOnly` and `TestReversingEntry` (test: run it)
- [ ] AC7: after the AC3 run, `docker compose exec -T postgres psql -U ledger -d ledger -tAc "select count(*) from pg_namespace where nspname like 'ledger_test_%'"` prints `0` (test: run it)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Out of scope
- Blocking privileged bypasses: `ALTER TABLE ... DISABLE TRIGGER`, `session_replication_role = replica`, or `DROP TRIGGER` by the table owner or a superuser. Separate DB roles with `REVOKE UPDATE, DELETE, TRUNCATE` for the app role belong with the migration-runner/roles work.
- A `reverses_entry_id` column, a link from a reversal to its original, or a Go `Reverse` function. A reversal here is just a new balanced entry.
- Mutable account attributes, such as status or closing an account.
- Append-only rules for future tables (outbox, idempotency records). Each of those needs its own decision, because the outbox relay updates rows to mark them published.
- Mapping SQLSTATE `23001` to an HTTP error, a migration runner, a strict mode that fails DB tests instead of skipping, and `.mex/` / README updates. Those are existing backlog items or the post-job mex update.
