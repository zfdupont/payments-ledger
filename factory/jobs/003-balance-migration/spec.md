# Add the first ledger migration with a deferred balance trigger

## Goal
Add `migrations/0001_ledger_core.up.sql` creating `accounts`, `journal_entries` and `journal_lines`. Postgres itself rejects, at commit time, any journal entry whose debits do not equal its credits in each currency. Add `internal/ledger/balance_test.go`, which applies that migration to a throwaway schema and proves an unbalanced entry fails at `COMMIT` and a balanced one commits. This job puts the first hard rule (balanced entries enforced in the database) into code and weakens none. `go test ./...` must still pass on a machine with no Postgres, so the DB test skips when `DATABASE_URL` is unset or the database is unreachable.

## Behavior
- With no `DATABASE_URL` in the environment, `go test ./...` exits 0 and `go test -v ./internal/ledger/` prints `--- SKIP: TestBalance` with the reason `DATABASE_URL not set; skipping Postgres test`.
- With `DATABASE_URL` set but nothing listening (for example port 5999), the test skips within about 3 s with `Postgres unreachable at DATABASE_URL: <error>`. It does not fail and does not hang.
- With the compose Postgres up (`docker compose up -d --wait postgres`) and `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable`, `go test -count=1 -v ./internal/ledger/` runs and passes these subtests:
  - `balanced`: USD debit 1000 and USD credit 1000. The commit succeeds and the entry row exists.
  - `balanced_multi_currency`: USD 1000/1000 plus EUR 50/50. The commit succeeds.
  - `unbalanced`: USD debit 1000 and USD credit 999. Each `INSERT` succeeds, because the check is deferred. `COMMIT` fails with SQLSTATE `23514` and the message `journal entry <id> is unbalanced in USD: debits 1000 <> credits 999`. Afterwards no `journal_entries` row with that id exists.
  - `one_sided`: a single USD debit of 1000. `COMMIT` fails with `23514`, message `... unbalanced in USD: debits 1000 <> credits 0`.
  - `balanced_total_not_per_currency`: USD debit 1000 and EUR credit 1000. `COMMIT` fails with `23514`, naming the first unbalanced currency in alphabetical order (`EUR: debits 0 <> credits 1000`).
  - `no_lines`: an entry header with no lines. `COMMIT` fails with `23514`, message `journal entry <id> has no lines`.
- The test leaves nothing behind in the database. Its `ledger_test_*` schema is dropped in cleanup.
- Row-level checks reject bad lines immediately at `INSERT`, not at commit: `amount_minor <= 0`, a `direction` other than `debit`/`credit`, or a `currency` that is not three uppercase letters. These checks are part of the schema. The required tests do not cover them.

## Design
The prototype below ran against the compose Postgres 18.7 with pgx v5.13.0, and all six subtests passed. The skip paths and `go vet ./...` were also checked. Keep this shape unless a check fails.

- `migrations/0001_ledger_core.up.sql` (new). The name follows the `{version}_{title}.up.sql` convention, so a migration runner can be adopted later without renaming. Plain SQL with no `BEGIN`/`COMMIT` (the runner or the test owns the transaction) and no schema qualification (objects go in the current `search_path`).
  - `accounts`: `id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, `name text NOT NULL UNIQUE`, `created_at timestamptz NOT NULL DEFAULT now()`.
  - `journal_entries`: `id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, `description text NOT NULL DEFAULT ''`, `created_at timestamptz NOT NULL DEFAULT now()`.
  - `journal_lines`: `id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, `entry_id bigint NOT NULL REFERENCES journal_entries (id)`, `account_id bigint NOT NULL REFERENCES accounts (id)`, `direction text NOT NULL CHECK (direction IN ('debit', 'credit'))`, `amount_minor bigint NOT NULL CHECK (amount_minor > 0)`, `currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$')`. Also add the index `journal_lines_entry_id_idx ON journal_lines (entry_id)`.
  - A plpgsql function `check_journal_entry_balanced() RETURNS trigger`:
    - It reads the entry id from `NEW.id` when `TG_TABLE_NAME = 'journal_entries'` and from `NEW.entry_id` otherwise. Use an `IF`, not a `CASE` expression.
    - If the entry has no lines: `RAISE EXCEPTION 'journal entry % has no lines', v_entry_id USING ERRCODE = 'check_violation'`.
    - Otherwise it selects the first currency, `ORDER BY currency LIMIT 1`, where `coalesce(sum(amount_minor) FILTER (WHERE direction = 'debit'), 0) <> coalesce(sum(amount_minor) FILTER (WHERE direction = 'credit'), 0)`. If one is found: `RAISE EXCEPTION 'journal entry % is unbalanced in %: debits % <> credits %', ... USING ERRCODE = 'check_violation'`.
    - Sums of `bigint` are `numeric`, so the comparison cannot overflow. The function returns `NULL`.
  - Two triggers, both `CREATE CONSTRAINT TRIGGER ... AFTER INSERT ... DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_journal_entry_balanced()`:
    - `journal_entries_balanced` on `journal_entries` (catches an entry with no lines).
    - `journal_lines_balanced` on `journal_lines` (catches unbalanced lines).
  - A header comment states what the migration creates and that balance is checked at commit.
- `migrations/0001_ledger_core.down.sql` (new). A comment-only, documented no-op: ledger tables are append-only records and are not dropped by a down migration. This follows `.mex/patterns/add-migration.md` step 2.
- `internal/ledger/balance_test.go` (new), `package ledger`. A test-only package is fine: `go build ./...` and `go vet ./...` both exit 0 with it. Contents:
  - A helper (for example `connect(t)`):
    - It skips if `os.Getenv("DATABASE_URL") == ""`.
    - Otherwise it calls `pgx.Connect` and `Ping` under a 3 s `context.WithTimeout`, and skips with `Postgres unreachable at DATABASE_URL: %v` on error.
    - It creates a schema named `ledger_test_<8 random hex chars>` (from `crypto/rand`) and runs `SET search_path TO <schema>`.
    - It reads `../../migrations/0001_ledger_core.up.sql` (`go test` runs in the package directory) and runs it with one `conn.Exec`. With no args, pgx uses the simple protocol, so a multi-statement file runs as one implicit transaction. A failure to apply is `t.Fatalf`, not a skip.
    - It seeds accounts `cash` and `revenue`.
    - It registers a `t.Cleanup` that runs `DROP SCHEMA <schema> CASCADE` and closes the connection, using `context.Background()`, because `t.Context()` is already cancelled when cleanups run.
  - A post helper:
    - It begins a tx, inserts a `journal_entries` row with `DEFAULT VALUES RETURNING id`, and inserts each line.
    - It returns the entry id, any insert error, and the `Commit` error separately, so the test can tell that the rejection came at commit.
    - It defers `tx.Rollback`.
  - `TestBalance`:
    - A table-driven test over the six cases in Behavior, with `t.Run` per case.
    - For each case it asserts that inserts returned no error.
    - Balanced cases: assert a nil commit error and a `journal_entries` count of 1 for the id.
    - Rejected cases: assert that `errors.As(commitErr, &*pgconn.PgError)` holds with `Code == "23514"`, and a count of 0 for the id. The prototype also logged the message with `t.Log`.
  - No other test files. The test does not write to the `public` schema.
- `go.mod` / `go.sum` (edit / new): `require github.com/jackc/pgx/v5 v5.13.0`, then `go mod tidy`. This is the project's first dependency and fills the "Postgres driver" TBD in `.mex/context/stack.md`. No query layer and no migration tool are added.
- Do not touch `docker-compose.yml`, `.env.example`, `factory/factory.yaml` or `cmd/ledger`. If the builder starts compose Postgres to verify, run `docker compose down` afterwards (keep the volume, so no `-v`).

## Acceptance criteria
- [ ] AC1: `migrations/0001_ledger_core.up.sql` creates `accounts`, `journal_entries` and `journal_lines`. `journal_lines.amount_minor` is `bigint` with `CHECK (amount_minor > 0)`, `journal_lines.currency` is `char(3)`, and no column uses a float or `numeric` money type (test: read the file; applied by `internal/ledger/balance_test.go`)
- [ ] AC2: the migration defines constraint triggers `journal_entries_balanced` and `journal_lines_balanced`, both `AFTER INSERT ... DEFERRABLE INITIALLY DEFERRED FOR EACH ROW`, executing `check_journal_entry_balanced()` (test: read the file)
- [ ] AC3: against compose Postgres, the `unbalanced`, `one_sided`, `balanced_total_not_per_currency` and `no_lines` subtests see every `INSERT` succeed and `COMMIT` fail with SQLSTATE `23514`, and the entry is not persisted (test: `internal/ledger/balance_test.go`, run with `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable go test -count=1 -v ./internal/ledger/` after `docker compose up -d --wait postgres`)
- [ ] AC4: in the same run, the `balanced` and `balanced_multi_currency` subtests commit without error and the entry row exists (test: `internal/ledger/balance_test.go`)
- [ ] AC5: with `DATABASE_URL` unset, `go test -count=1 -v ./internal/ledger/` exits 0 and prints `--- SKIP: TestBalance`. With `DATABASE_URL=postgres://ledger:changeme@localhost:5999/ledger?sslmode=disable` it also exits 0 with a skip, within 5 s (test: `internal/ledger/balance_test.go`, run both)
- [ ] AC6: after the AC3 run, `docker compose exec -T postgres psql -U ledger -d ledger -tAc "select count(*) from pg_namespace where nspname like 'ledger_test_%'"` prints `0` (test: run it)
- [ ] AC7: `migrations/0001_ledger_core.down.sql` exists and contains only comments explaining that ledger tables are not dropped (test: read the file)
- [ ] AC8: `go.mod` requires `github.com/jackc/pgx/v5` and `go.sum` is committed. No other direct dependency is added (test: read `go.mod`)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Out of scope
- Append-only enforcement (UPDATE/DELETE/TRUNCATE triggers) on ledger tables. That is the next backlog item, which also covers balance bypass via UPDATE or DELETE of lines.
- Enforcing that a line's currency matches its account's currency (accounts carry no currency yet), plus account types and normal balances.
- Choosing or wiring a migration runner, a `migrate` command, a `schema_migrations` table, or setting the migration command in `.mex/context/setup.md`.
- Go ledger service code (non-test files in `internal/ledger`), transfers, idempotency, the outbox and the HTTP API.
- Making the test fail instead of skip when Postgres is missing, for example in CI. The CI backlog job can add an opt-in.
- Updating `.mex/`, `CLAUDE.md` or the README. The post-job mex update handles those.
