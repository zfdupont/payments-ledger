VERDICT: PASS

Reviewed HEAD 7a0961d against main. commands.md was recorded at the same sha, so I did not rerun anything. Build, vet, and test pass with and without DATABASE_URL.

Checklist:

1. Balance enforced in the database: pass. `migrations/0001_ledger_core.up.sql` is unchanged (empty diff against main). Its deferred constraint triggers still fire on INSERT, and 0002 does not touch them. The new triggers also close the UPDATE/DELETE bypass of `journal_lines` that job 003 left open.
2. Append-only: pass. `migrations/0002_ledger_append_only.up.sql:25-42` puts a `FOR EACH STATEMENT` `BEFORE UPDATE OR DELETE OR TRUNCATE` trigger on `accounts`, `journal_entries` and `journal_lines`. It raises `restrict_violation` (23001). Statement-level triggers also catch zero-row updates and TRUNCATE. `ON CONFLICT DO UPDATE` is covered, and the test has a subtest for it. Corrections are reversing INSERTs, and `TestReversingEntry` shows one commits with net balances of 0.
3. Idempotency: not applicable. The diff adds no write endpoint and no non-test Go code.
4. Outbox and events: not applicable. No handler or publisher changes.
5. Money: the diff introduces no floats. The test amounts are int64 minor units with a currency code.
6. Migrations: pass. 0002 is a new versioned pair, and 0001 is not edited. The down migration is comment-only, as `.mex/patterns/add-migration.md` requires. It contains no BEGIN/COMMIT and no schema qualification, matching 0001. The function reads no tables, so the open search_path follow-up from job 003 does not apply.

Test review:
- `internal/ledger/balance_test.go` changes only the `connect` helper. It globs `../../migrations/*.up.sql`, sorts the files with `slices.Sort`, and applies each in turn. It fails the test if no files match or any file fails to apply.
- `internal/ledger/append_only_test.go` asserts SQLSTATE 23001 and the exact message for each (table, operation) pair. It then checks that row counts and values are unchanged. It also has the extra `ON CONFLICT DO UPDATE` subtest, which the spec's Behavior section covers.

No defects found. Privileged bypasses (`DISABLE TRIGGER`, `session_replication_role`, `DROP TRIGGER` by the owner) are explicitly out of scope in the spec and are documented in the migration header.
