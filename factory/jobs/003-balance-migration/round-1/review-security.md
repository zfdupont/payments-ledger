VERDICT: PASS

Round 1 security review of factory/003-balance-migration (HEAD 69a8ab4). Commands at that sha all pass per commands.md.

The diff has no HTTP endpoints, auth middleware, token validation, API key storage or idempotency code. It adds a migration, a DB test and the pgx dependency. The checklist items that need such code do not apply to this job. Findings on the items that do apply:

1. Write endpoints and auth: not applicable. No routes were added or changed, and `cmd/ledger` is untouched.
2. Token validation (issuer, audience, expiry, signature): not applicable. No token code is in the diff.
3. API key hashing and constant-time compare: not applicable. No key storage or comparison is in the diff.
4. Idempotency keys scoped per client: not applicable. The schema has no idempotency table yet. When that table is added, its uniqueness must include the client id.
5. Secrets in code, logs or errors: clean.
   - No credentials, tokens or `.env` content are in the diff. `balance_test.go` reads `DATABASE_URL` only from the environment.
   - The skip message `Postgres unreachable at DATABASE_URL: %v` prints the pgx connect error. That error names host, user and database, not the password.
   - The `changeme` placeholder appears only in the spec and commands.md, not in committed code.
   - `go.sum` holds module hashes only.
6. SQL parameterization: clean.
   - `balance_test.go:52` and `balance_test.go:64` build SQL with string concatenation, `"DROP SCHEMA IF EXISTS "+schema+" CASCADE"`, `"CREATE SCHEMA "+schema` and `"SET search_path TO "+schema`. The `schema` value is `ledger_test_` plus `hex.EncodeToString` of 4 `crypto/rand` bytes, so no request or env input reaches it. Identifiers cannot be parameterized, so this is acceptable.
   - All value inserts and selects in `post` and `TestBalance` use `$n` parameters.
   - The migration is static DDL. In `check_journal_entry_balanced()` the entry id comes from `NEW`, and the function uses no dynamic `EXECUTE`.
7. Function privileges: `check_journal_entry_balanced()` is not `SECURITY DEFINER`. It runs with the caller's rights and has no privilege-escalation or `search_path` hijack surface.

Notes, not defects. Both are explicitly out of scope per spec.md.
- Balance bypass through UPDATE or DELETE of lines, and TRUNCATE, is not blocked here. The next backlog job (append-only triggers) covers it. Until then, a line can be changed or removed after commit without re-checking balance.
- Line currency is not tied to the account's currency.
