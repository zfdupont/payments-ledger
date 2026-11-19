---
name: conventions
description: How code is written in this project — naming, structure, patterns, and style. Load when writing new code or reviewing existing code.
triggers:
  - "convention"
  - "pattern"
  - "naming"
  - "style"
  - "how should I"
  - "what's the right way"
edges:
  - target: context/architecture.md
    condition: when a convention depends on understanding the system structure
  - target: context/decisions.md
    condition: when a convention's reasoning is needed
  - target: patterns/INDEX.md
    condition: when a task matches a pattern
# Add only nodes that embody the documented convention; do not ground examples broadly.
# grounds_to:
#   - node: "function:<tier-1-id>"
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
  type: convention
  status: promoted
  revision: 3
  title: conventions
  relations:
    - type: related_to
      target: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
      note: when a convention depends on understanding the system structure
    - type: related_to
      target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
      note: when a convention's reasoning is needed
---

# Conventions

<!-- Read broad, ground tight. Anchor concrete symbols while keeping prose readable:
```markdown
[`someFunction()`](mex://function:<tier-1-id>)
```
-->

## Naming
- Database tables and columns: snake_case. [VERIFY AFTER FIRST IMPLEMENTATION]
- Money fields: `amount_minor` (int64) paired with `currency` (ISO 4217)
- Go packages: binaries under `cmd/<name>` (currently `cmd/ledger`); internal code planned under `internal/<area>` (e.g. `internal/ledger`, `internal/api`)

## Structure
- Ledger table writes live only in the ledger service package; handlers never write ledger tables
- Outbox rows are written in the same transaction as the ledger change they describe
- Schema changes only through versioned migration files
- Layout: `cmd/ledger` (entry point), `migrations/` (planned), `internal/` (planned), `docker-compose.yml` at repo root

## Patterns
Idempotent writes: check-or-insert the idempotency record inside the same transaction as the write.
```
// Correct: one tx -> idempotency record + journal lines + outbox row
// Wrong: write ledger, then publish to Kafka from the handler
```

Corrections: post a reversing journal entry; never UPDATE or DELETE ledger rows.

## Verify Checklist
Before presenting any code:
- [ ] Every new write endpoint requires and honors an idempotency key
- [ ] Journal entries balance, and the database rejects unbalanced ones
- [ ] No UPDATE/DELETE on ledger rows
- [ ] Events go through the outbox in the same transaction
- [ ] Money uses integer minor units plus currency
- [ ] Schema changes are in a new versioned migration
- [ ] Temporal activities are idempotent and safe to retry
- [ ] `go test ./...` and `go vet ./...` pass
