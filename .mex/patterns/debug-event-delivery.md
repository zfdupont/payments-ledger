---
name: debug-event-delivery
description: Diagnosing missing, duplicated, or stuck events across the outbox relay, Redpanda, and Temporal retries.
triggers:
  - "missing event"
  - "duplicate event"
  - "outbox"
  - "workflow stuck"
  - "kafka"
edges:
  - target: context/architecture.md
    condition: for the outbox and workflow flow
  - target: patterns/add-idempotent-write-endpoint.md
    condition: when the root cause is in the write path
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ8TVDDDEW9TNM9GWZWSKN
  type: pattern
  status: promoted
  revision: 3
  title: debug-event-delivery
  relations:
    - type: related_to
      target: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
      note: for the outbox and workflow flow
    - type: related_to
      target: mx_01M7SQ8TRNA59FNBP896Y01227
      note: when the root cause is in the write path
---

# Debug Event Delivery

## Context
Delivery is at-least-once: outbox row in the ledger tx, relay publishes to Redpanda, Temporal retries activities. [VERIFY AFTER FIRST IMPLEMENTATION]

## Steps
1. Find the ledger change and confirm its outbox row exists.
2. Check whether the relay marked it published; if not, check relay logs and broker connectivity (`KAFKA_BROKERS`).
3. Consume the topic and look for the event by id.
4. For workflow-driven posts, inspect the workflow history in Temporal for failed or retrying activities.

## Gotchas
- Duplicates are expected; fix the consumer's dedup, not the relay.
- A non-idempotent activity can double-post on retry.

## Verify
- [ ] Root cause identified at a specific boundary
- [ ] A test reproduces it

## Debug
- Relay stuck on one row: check for a poison message and ordering assumptions.

## Update Scaffold
- [ ] Update `.mex/ROUTER.md` "Current Project State" if what's working/not built has changed
- [ ] Update any `.mex/context/` files that are now out of date
- [ ] If this is a new task type without a pattern, create one in `.mex/patterns/` and add to `INDEX.md`
