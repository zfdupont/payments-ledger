VERDICT: PASS

Round 1, HEAD 69a8ab4. HTTP API contract review.

1. No HTTP surface in this diff. The change set is migrations/0001_ledger_core.up.sql, migrations/0001_ledger_core.down.sql, internal/ledger/balance_test.go, go.mod and go.sum. There are no handlers, routes, request or response shapes, or Idempotency-Key handling. The spec puts the HTTP API, idempotency and the outbox out of scope. So the idempotency, status code, error body, breaking change and over-HTTP test checks do not apply, and nothing existing is broken.

2. Money representation, which this diff does bear on, matches the contract. `journal_lines.amount_minor` is `bigint` with `CHECK (amount_minor > 0)`, and `currency` is `char(3)` with `CHECK (currency ~ '^[A-Z]{3}$')`. The only `numeric` use is the local sum variables `v_debits` and `v_credits` in the trigger function (up.sql around line 57). They avoid overflow and are not stored columns. No floats.

3. Forward note for later API jobs, not a defect. The trigger raises SQLSTATE 23514 with the messages `journal entry <id> is unbalanced in <CCY>: ...` and `journal entry <id> has no lines`. When the HTTP layer is added, it should map these to a stable client error, for example 422 with a typed error code. It should not return the raw Postgres message as the error body.

Commands (commands.md at 69a8ab4): build, test (with and without DATABASE_URL) and vet all exit 0.
