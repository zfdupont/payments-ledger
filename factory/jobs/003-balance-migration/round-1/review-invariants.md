VERDICT: PASS

Reviewed base...HEAD at 69a8ab49eb4a175025f7c62be7fd2e7f9a9fa145, which matches the HEAD recorded in commands.md. All recorded commands exit 0: build, vet, and the test run against compose Postgres (6/6 subtests) and with no DATABASE_URL.

No blocking defects. Checklist:

1. Balance enforced in the database: PASS. `migrations/0001_ledger_core.up.sql:70-78` defines `journal_entries_balanced` and `journal_lines_balanced` as `CONSTRAINT TRIGGER ... AFTER INSERT ... DEFERRABLE INITIALLY DEFERRED FOR EACH ROW`. Both call `check_journal_entry_balanced()` (lines 33-68). The function raises `check_violation` (23514) on an entry with no lines (line 46). It also raises 23514 on any currency where debits differ from credits (HAVING at lines 56-57, raise at lines 62-64). The check is per currency, and the sums are numeric so they cannot overflow. A line inserted later against an already-committed entry re-fires the check on that entry, so appending to a committed entry cannot unbalance it. `TestBalance` shows the inserts succeed and COMMIT fails with 23514 and the exact messages, and that the rejected entry row is not persisted. The first-unbalanced-currency order is alphabetical (`ORDER BY currency`).

2. Append-only: PASS for this job's scope. The migration and test add no UPDATE or DELETE on ledger tables. The test only INSERTs and runs SELECT count. The test's `DROP SCHEMA` touches only its own throwaway schema. Triggers for UPDATE, DELETE and TRUNCATE are explicitly out of scope per spec and are the next backlog item. Until then, balance can be bypassed by UPDATE or DELETE of lines, which the spec acknowledges.

3. Idempotency: not applicable. No write endpoint is added or changed.

4. Outbox and events: not applicable. No handler, event or Kafka code is added.

5. Money: PASS. `journal_lines.amount_minor` is `bigint` with `CHECK (amount_minor > 0)` (up.sql line 29). `currency` is `char(3)` with `CHECK (currency ~ '^[A-Z]{3}$')` (line 30). No float or numeric column exists. `numeric` appears only in plpgsql local variables used for message formatting and the comparison. The test uses int64.

6. Migrations: PASS. Only new files `0001_ledger_core.up.sql` and `0001_ledger_core.down.sql`; no existing migration is edited. The down migration is comments only, per `.mex/patterns/add-migration.md`. There is no BEGIN/COMMIT and no schema qualification. The test applies the file into a throwaway `ledger_test_<hex>` schema and drops it in `t.Cleanup` using `context.Background()`.

Non-blocking notes, no change required:
- `check_journal_entry_balanced()` resolves `journal_lines` through the session `search_path` and is not pinned with `SET search_path`. This is intentional per spec, because the test relies on a throwaway schema. Consider pinning it when a migration runner is adopted.
- The go.mod diff adds only pgx as a direct dependency. The indirect additions are pgpassfile, pgservicefile and x/text.
