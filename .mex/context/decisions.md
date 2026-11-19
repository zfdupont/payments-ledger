---
name: decisions
description: Key architectural and technical decisions with reasoning. Load when making design choices or understanding why something is built a certain way.
triggers:
  - "why do we"
  - "why is it"
  - "decision"
  - "alternative"
  - "we chose"
edges:
  - target: context/architecture.md
    condition: when a decision relates to system structure
  - target: context/stack.md
    condition: when a decision relates to technology choice
  - target: context/conventions.md
    condition: when a decision becomes a coding rule
# Decisions usually ground sparsely; add only symbols that implement the decision.
# Entry shape: { node: "function:<tier-1-id>" } (MEX records the fingerprint)
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
  type: architecture
  status: promoted
  revision: 4
  title: decisions
  relations:
    - type: related_to
      target: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
      note: when a decision relates to system structure
    - type: related_to
      target: mx_01M7SQ5NGJ7NVKAB5BYQW34E29
      note: when a decision relates to technology choice
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: when a decision becomes a coding rule
---

# Decisions

<!-- If a decision names its concrete implementation point, link it as below;
     do not anchor vague concepts:
```markdown
[`someFunction()`](mex://function:<tier-1-id>)
```
-->

<!-- HOW TO USE THIS FILE:
     Each decision follows the format below.
     When a decision changes: DO NOT delete the old entry.
     Mark it as superseded, add the new entry above it.
     The history must be preserved — this is the event clock. -->

## Decision Log

<!-- mex:entity
id: mx_01M7SQ8TNNZ3MMJFW6SVJCBYQ7
type: decision
status: promoted
revision: 2
relations:
  - type: refines
    target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
    note: Section of decisions in context/decisions.md
-->
### Build one app in phases
**Date:** 2026-11-19
**Status:** Active
**Decision:** Build the ledger as one service in phases so later Kubernetes/EKS work deploys a real service.
**Reasoning:** Infra work on a throwaway app demonstrates less; one app also gives a single portfolio entry targeting infra and fintech roles.
**Alternatives considered:** Separate demo apps per skill (rejected: shallow, no real deploy target).
**Consequences:** Phase 1 must be deployable as-is later; keep config in env vars.

<!-- mex:entity
id: mx_01M7SQ8TM0M8MB524J3MR99Z41
type: decision
status: promoted
revision: 2
relations:
  - type: refines
    target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
    note: Section of decisions in context/decisions.md
-->
### Transactional outbox instead of dual writes
**Date:** 2026-11-19
**Status:** Active
**Decision:** Ledger changes and their events are written in one Postgres transaction; a relay publishes to Redpanda.
**Reasoning:** Dual writes can commit the ledger and lose the event, or the reverse.
**Alternatives considered:** Publish from handler after commit (rejected: lost events), CDC/Debezium (rejected: heavier for Phase 1).
**Consequences:** Delivery is at-least-once; consumers must deduplicate.

<!-- mex:entity
id: mx_01M7SQ8TJM6V9H1JC9GYHKPM7S
type: decision
status: promoted
revision: 2
relations:
  - type: refines
    target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
    note: Section of decisions in context/decisions.md
-->
### Redpanda for the Kafka API
**Date:** 2026-11-19
**Status:** Active
**Decision:** Use Redpanda as the broker, coding against the Kafka API.
**Reasoning:** Kafka-compatible with a lighter local footprint on Docker Compose.
**Alternatives considered:** Apache Kafka (heavier locally; remains drop-in via the same API).
**Consequences:** Use only Kafka-API features so the broker stays swappable.

<!-- mex:entity
id: mx_01M7SQ8TH4YWRKCJD607KFNW5T
type: decision
status: promoted
revision: 2
relations:
  - type: refines
    target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
    note: Section of decisions in context/decisions.md
-->
### Temporal for retries and reconciliation
**Date:** 2026-11-19
**Status:** Active
**Decision:** Durable retries and reconciliation run as Temporal workflows on a self-hosted dev server.
**Reasoning:** Durable workflow state instead of ad hoc retry loops and cron jobs.
**Alternatives considered:** [TO BE DETERMINED: record alternatives if revisited]
**Consequences:** Activities must be idempotent.

<!-- mex:entity
id: mx_01M7SQ8TEXQH1B57KP90YE5SV1
type: decision
status: promoted
revision: 2
relations:
  - type: refines
    target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
    note: Section of decisions in context/decisions.md
-->
### Phase 1 runs locally at $0
**Date:** 2026-11-19
**Status:** Active
**Decision:** Phase 1 runs entirely on Docker Compose.
**Reasoning:** No cloud spend until the service exists.
**Alternatives considered:** Cloud-managed Postgres/Kafka (rejected for Phase 1: cost).
**Consequences:** EKS and managed services arrive in a later phase.

