VERDICT: PASS

Scope: the diff at HEAD 7a0961d adds migration 0002 and ledger tests. It has no HTTP surface. I found no handlers, routes, request or response shapes, or Idempotency-Key code in the diff or in cmd/ and internal/. The spec also leaves HTTP mapping of SQLSTATE 23001 out of scope. The checklist items are handled as follows.

1. Idempotency-Key handling: not applicable. No endpoint is added or changed. Nothing in the diff touches idempotency records, and the spec's Out of scope section says they get their own later decision.
2. Status codes and error bodies: not applicable. No HTTP error is produced. The DB error contract is SQLSTATE 23001, message `<table> is append-only: <OP> is not allowed`, and a fixed hint. That contract is pinned exactly in internal/ledger/append_only_test.go (the `pgErr.Code` and `pgErr.Message` assertions). It gives a later HTTP layer a stable value to map. That mapping is explicitly deferred to the backlog.
3. Amounts: the new test lines use `amount_minor` as int64 with a `currency` code (`namedLine` in append_only_test.go). That matches the integer-minor-units rule.
4. Breaking changes to existing request or response shapes: none. The only existing-code edit is the `connect` test helper in internal/ledger/balance_test.go. It now applies every `migrations/*.up.sql` in sorted order. No API shape changes.
5. Contract tests over HTTP for new endpoints: no new endpoints, so nothing is required.

Future note, not a defect: when the HTTP write path is built, a handler that sees 23001 must not surface it as a 5xx with the raw message. The mapping belongs in the backlog item the spec names.

Commands (commands.md at HEAD 7a0961d): build, test (with and without DATABASE_URL) and vet all exit 0.
