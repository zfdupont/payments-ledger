---
name: router
description: Session bootstrap and navigation hub. Read at the start of every session before any task. Contains project state, routing table, and behavioural contract.
edges:
  - target: context/architecture.md
    condition: when working on system design, integrations, or understanding how components connect
  - target: context/stack.md
    condition: when working with specific technologies, libraries, or making tech decisions
  - target: context/conventions.md
    condition: when writing new code, reviewing code, or unsure about project patterns
  - target: context/decisions.md
    condition: when making architectural choices or understanding why something is built a certain way
  - target: context/setup.md
    condition: when setting up the dev environment or running the project for the first time
  - target: patterns/INDEX.md
    condition: when starting a task — check the pattern index for a matching pattern file
last_updated: 2026-11-19
---

# Session Bootstrap

If you haven't already read `AGENTS.md`, read it now — it contains the project identity, non-negotiables, and commands.

Then read this file fully before doing anything else in this session.

## Current Project State
**Working:**
- Nothing yet. New project: repo scaffold, CLAUDE.md, mex scaffold, and factory config only.

**Not yet built (Phase 1, ledger core):**
- Go module, HTTP API, and Docker Compose stack (Postgres, Redpanda, Temporal dev server)
- Double-entry ledger schema and versioned migrations
- Idempotency keys on write endpoints
- Transactional outbox and relay to Redpanda
- Temporal retry and reconciliation workflows
- Rate-limited external ingestion with retries and dedup
- OAuth2 client credentials and API key auth
- `EXPLAIN` slow-query tuning write-up

**Later phases:**
- Kubernetes deployment on EKS

**Known issues:**
- None yet.

## Routing Table

Load the relevant file based on the current task. Always load `context/architecture.md` first if not already in context this session.

| Task type | Load |
|-----------|------|
| Understanding how the system works | `context/architecture.md` |
| Working with a specific technology | `context/stack.md` |
| Writing or reviewing code | `context/conventions.md` |
| Making a design decision | `context/decisions.md` |
| Setting up or running the project | `context/setup.md` |
| Adding a ledger write endpoint | `patterns/add-idempotent-write-endpoint.md` |
| Changing the database schema | `patterns/add-migration.md` |
| Missing, duplicate, or stuck events | `patterns/debug-event-delivery.md` |
| Any specific task | Check `patterns/INDEX.md` for a matching pattern |

## Behavioural Contract

For every task, follow this loop:

1. **CONTEXT** — Load the relevant context file(s) from the routing table above. Check `patterns/INDEX.md` for a matching pattern. If one exists, follow it.
2. **BUILD** — Do the work. If a pattern exists, follow its Steps. If you are about to deviate from an established pattern, say so before writing any code — state the deviation and why.
3. **VERIFY** — Load `context/conventions.md` and run the Verify Checklist item by item. State each item and whether the output passes. Do not summarise — enumerate explicitly.
4. **DEBUG** — If verification fails or something breaks, check `patterns/INDEX.md` for a debug pattern. Follow it. Fix the issue and re-run VERIFY.
5. **GROW** — After meaningful work, run this binary checklist:
   - **Ground:** What changed in reality? Name the changed behavior, system, command, dependency, or workflow.
   - **Record:** If project state changed, update the "Current Project State" section above. If documented facts changed, update the relevant `context/` file surgically. Record a new behavioral claim about specific code as its own entity (`mex wiki apply` with a `create-entry` operation), grounded to every symbol whose change could make it false and naming the identifiers and literals it depends on. Ground with node ids only; MEX records the fingerprints. Give the entity 3 to 6 `aliases` (short plain-word phrases a developer would type for the symptom or outcome it covers, written from the claim itself, not its title or identifiers).
   - **Connect:** Link section entities to their enclosing knowledge with `refines` using existing ids. Preview missing structural links with `mex wiki link-sections`, then apply explicitly. Review semantic relationships from evidence; do not connect records just because they share keywords or code groundings.
   - **Orient:** If this task can recur and no pattern exists, create one in `patterns/` using `patterns/README.md`, then add it to `patterns/INDEX.md`. If a pattern exists but you learned a gotcha, update it.
   - **Write:** Bump `last_updated` in every scaffold file you changed. Read `mex logging --json` before optional `mex log` notes: `significant` records material rationale, `checkpoints` batches useful notes at task/session boundaries, and `manual` avoids unsolicited notes. Honor explicit user log requests in every mode; mandatory workflow Activity and recovery audits remain required.
