VERDICT: PASS

No defects found. Reviewed HEAD 69a8ab4 against the spec. I did not rerun the commands, because commands.md records the same sha, and I used it as authoritative (build, vet, test with and without DATABASE_URL all exit 0, 6/6 subtests pass with exact messages).

Checked independently:
1. `migrations/0001_ledger_core.up.sql`: matches the spec (AC1, AC2).
   - `amount_minor` is bigint with CHECK > 0.
   - `currency` is char(3) with a regex CHECK.
   - Both constraint triggers are AFTER INSERT, DEFERRABLE INITIALLY DEFERRED, FOR EACH ROW.
   - The function raises `check_violation` (23514).
   - Sums are numeric, so they cannot overflow.
2. I probed a throwaway schema on compose Postgres and dropped it afterwards.
   - amount 0, a bad direction and lowercase currency are each rejected immediately at INSERT by the row CHECKs.
   - Four lines of 9223372036854775807 (two debits, two credits) commit with no overflow.
   - A header-only insert in autocommit fails with "has no lines".
3. `go mod tidy` produces no diff, so go.mod and go.sum are consistent. pgx v5.13.0 is the only direct dependency (AC8).
4. The `pg_namespace ... like 'ledger_test_%'` count is 0 after the run (AC6).
5. `internal/ledger/balance_test.go` follows the spec.
   - It skips on an unset or unreachable DATABASE_URL, using a 3 s timeout and the exact skip messages.
   - The cleanup uses `context.Background()`.
   - It asserts that inserts succeed, that the commit error is a PgError with 23514 and the exact message, and that the entry count is 0 after rollback.
   - The migration-apply failure is `t.Fatalf`, not a skip.
6. The down migration is comments only (AC7).

Non-blocking notes, no action required:
- The builder used `GROUP BY ... HAVING ... ORDER BY currency LIMIT 1` instead of the spec's wording. It is equivalent, and alphabetical-first currency ordering is verified by the `balanced_total_not_per_currency` subtest.
- Per-row triggers re-check the entry once per line, which is O(n^2) for large entries. This is acceptable for this job and not a correctness issue.
- Bypass through UPDATE or DELETE of lines is explicitly out of scope and belongs to the append-only backlog item.
