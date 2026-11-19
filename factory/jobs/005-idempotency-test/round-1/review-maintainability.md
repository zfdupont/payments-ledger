VERDICT: PASS

Reviewed d0923cc (matches commands.md HEAD; build, test with and without DATABASE_URL, race and vet all exit 0).

Checklist results:
- Ledger writes only in the ledger service: the only INSERTs into journal_entries and journal_lines are in internal/ledger/ledger.go. internal/api/journal_entries.go calls ledger.PostEntry and writes no SQL. The only SQL in internal/api non-test code is on idempotency_keys, which is not a ledger table. No Kafka or outbox code is in internal/api.
- Naming: snake_case tables and columns in migrations/0003_idempotency_keys.up.sql. Money is int64 `amount_minor` plus `currency` in the request structs and ledger.Line. The new Go package layout (internal/api, internal/ledger) matches conventions.md.
- Pattern: follows .mex/patterns/add-idempotent-write-endpoint.md. The idempotency record is inserted in the same transaction as the write, and a primary key with ON CONFLICT DO NOTHING settles races instead of read-then-write. Steps 2 (auth) and 5 (outbox) are deferred, and the spec states that explicitly. The pattern's "[VERIFY AFTER FIRST IMPLEMENTATION: package paths]" marker is for the post-job mex step.
- Dead code and speculation: nothing unused. `Router.replayCommitted` and the non-2xx branch in `idempotent` are both required by the spec. The only structure beyond the minimum is `addWrite`, which the spec asks for.
- Env vars: none added, so `.env.example` and CLAUDE.md need no change.
- Migrations: 0001 and 0002 are untouched. 0003 has up and down files, and the header comments follow the 0001/0002 style. cmd/, go.mod and go.sum are untouched.
- .mex scaffold: it now lags the code in these places:
  - The "No code yet" Commands section in CLAUDE.md.
  - The router TBD in .mex/context/stack.md.
  - The VERIFY marker and package paths in the pattern file.
  - The "HTTP package is planned" line in conventions.md.
  - The idempotency_keys table in architecture.
  The spec assigns this to the post-job mex step (see the CLAUDE.md Workflow section), so it is not a defect in this diff.

Non-blocking notes (no action required for PASS):
1. internal/api/idempotency.go:131-137 (the concurrent-conflict path through `replayCommitted`) has no test. The spec lists only sequential cases, and build.md says so. If a later job touches this layer, add a two-connection test.
2. internal/api/idempotency.go:76-80 (`writeRaw`) ignores the `w.Write` error. This is conventional for handlers and vet is clean.
3. internal/api/idempotency.go:122-125 (the non-2xx handler branch) is unreachable with the single route today. It is spec-mandated and should be kept.
