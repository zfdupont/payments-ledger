---
name: add-idempotent-write-endpoint
description: Adding a write endpoint that changes the ledger (account, journal entry, transfer) with idempotency and outbox.
triggers:
  - "add endpoint"
  - "transfer"
  - "journal entry"
  - "idempotency"
edges:
  - target: context/conventions.md
    condition: always, for the verify checklist
  - target: context/architecture.md
    condition: when unsure how the write path flows to the outbox
  - target: patterns/add-migration.md
    condition: when the endpoint needs new tables or constraints
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ8TRNA59FNBP896Y01227
  type: pattern
  status: promoted
  revision: 4
  title: add-idempotent-write-endpoint
  relations:
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: always, for the verify checklist
    - type: related_to
      target: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
      note: when unsure how the write path flows to the outbox
    - type: related_to
      target: mx_01M7SQ8TT1DDW3YSVG242JYZ38
      note: when the endpoint needs new tables or constraints
---

# Add an Idempotent Write Endpoint

## Context
Load `context/architecture.md` (write path) and `context/conventions.md`. All ledger writes go through the ledger service. [VERIFY AFTER FIRST IMPLEMENTATION: package paths]

## Steps
1. Require the `Idempotency-Key` header; reject the request if it is missing.
2. Authenticate (OAuth2 client credentials or API key) and scope the key to the client.
3. In one Postgres transaction: insert-or-fetch the idempotency record; on a replay with the same request hash, return the stored response; with a different hash, return a conflict.
4. Write balanced journal lines through the ledger service.
5. Insert the outbox row describing the change.
6. Store the response on the idempotency record and commit.
7. Add tests for first call, replay, conflicting replay, and unbalanced input.

## Gotchas
- Concurrent requests with the same key: rely on a unique constraint, not a read-then-write check.
- Never publish to Kafka from the handler.
- Amounts are int64 minor units; validate currency.

## Verify
- [ ] Replay returns the identical response and writes nothing new
- [ ] Unbalanced entries are rejected by the database
- [ ] Exactly one outbox row per committed change

## Debug
- Duplicate ledger rows: check the idempotency unique constraint and that the record is in the same tx.
- Missing events: check the outbox row was written in the tx, then the relay.

## Update Scaffold
- [ ] Update `.mex/ROUTER.md` "Current Project State" if what's working/not built has changed
- [ ] Update any `.mex/context/` files that are now out of date
- [ ] If this is a new task type without a pattern, create one in `.mex/patterns/` and add to `INDEX.md`
