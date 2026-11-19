---
name: add-migration
description: Adding a versioned Postgres schema migration, including ledger invariant constraints.
triggers:
  - "migration"
  - "schema"
  - "add table"
  - "add column"
edges:
  - target: context/conventions.md
    condition: for naming and money-type rules
  - target: context/setup.md
    condition: for the migration command
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ8TT1DDW3YSVG242JYZ38
  type: pattern
  status: promoted
  revision: 3
  title: add-migration
  relations:
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: for naming and money-type rules
    - type: related_to
      target: mx_01M7SQ5NGEAW9G41KJ3AV8A9GE
      note: for the migration command
---

# Add a Migration

## Context
Schema changes only through versioned migrations. [VERIFY AFTER FIRST IMPLEMENTATION: migration tool and directory]

## Steps
1. Create a new migration with the next version; never edit an applied one.
2. Write up and down steps (down may be a no-op for ledger tables, documented).
3. Enforce invariants in the schema: balanced entries, append-only ledger tables, unique idempotency keys.
4. Apply against a fresh local database and run tests.

## Gotchas
- Balance checks across lines need a deferred constraint trigger, not a row CHECK.
- Append-only: revoke UPDATE/DELETE or add a trigger that rejects them.
- Money columns are bigint minor units plus a currency column.

## Verify
- [ ] Migration applies cleanly to an empty database
- [ ] Existing applied migrations are unchanged
- [ ] Invariants are enforced by the database

## Debug
- Migration fails: check `DATABASE_URL` and that the compose Postgres is up.

## Update Scaffold
- [ ] Update `.mex/ROUTER.md` "Current Project State" if what's working/not built has changed
- [ ] Update any `.mex/context/` files that are now out of date
- [ ] If this is a new task type without a pattern, create one in `.mex/patterns/` and add to `INDEX.md`
