VERDICT: PASS

Round 1 security review of factory/004-append-only (7a0961d). Commands in commands.md all exit 0 at the same HEAD.

The diff has four files: migration 0002 up and down, the `connect` helper in balance_test.go, and append_only_test.go. It adds no HTTP handlers, middleware, token or API key code, or idempotency code, so the auth items on the checklist do not apply.

1. Write endpoint auth: not applicable. The diff has no routes and does not touch `cmd/`.
2. Token validation (issuer, audience, expiry, signature): not applicable. No token code was added or changed.
3. API key hashing and constant-time compare: not applicable.
4. Idempotency key scoping per client: not applicable. The job adds no idempotency table or logic, and the spec leaves that to a later decision.
5. Secrets in code, logs, errors or committed files: clean.
   - The diff has no credentials, tokens or `.env` content.
   - The tests read `DATABASE_URL` only from the environment, and no new log or error path prints it.
   - The spec's compose DSN appears only in the spec and docs, not in the diff.
6. SQL parameters: clean.
   - The migration is static DDL.
   - The test SQL is fixed literals, and the variable values (`id`, description, account name, direction, amount, currency) are passed as `$n` parameters. Nothing is built from request input.
   - `reject_ledger_mutation()` is `plpgsql` without `SECURITY DEFINER`. It reads no tables and interpolates only `TG_TABLE_NAME` and `TG_OP` through RAISE format placeholders, so it has no injection or search_path exposure.
7. Integrity effect (positive): the statement-level triggers also close the balance-trigger bypass through UPDATE or DELETE of `journal_lines`. `INSERT ... ON CONFLICT DO UPDATE` is covered by a test. The migration header documents that privileged bypasses (`DISABLE TRIGGER`, `session_replication_role`, owner `DROP TRIGGER`) are out of scope and belong with the DB roles work, as the spec says.

No defects found.
