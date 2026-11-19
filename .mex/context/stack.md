---
name: stack
description: Technology stack, library choices, and the reasoning behind them. Load when working with specific technologies or making decisions about libraries and tools.
triggers:
  - "library"
  - "package"
  - "dependency"
  - "which tool"
  - "technology"
edges:
  - target: context/decisions.md
    condition: when the reasoning behind a tech choice is needed
  - target: context/conventions.md
    condition: when understanding how to use a technology in this codebase
  - target: context/setup.md
    condition: when installing or running these technologies locally
# Broad inventory: ground only claims embodied by a small number of symbols.
# Entry shape: { node: "function:<tier-1-id>" } (MEX records the fingerprint)
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ5NGJ7NVKAB5BYQW34E29
  type: architecture
  status: promoted
  revision: 4
  title: stack
  relations:
    - type: related_to
      target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
      note: when the reasoning behind a tech choice is needed
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: when understanding how to use a technology in this codebase
    - type: related_to
      target: mx_01M7SQ5NGEAW9G41KJ3AV8A9GE
      note: when installing or running these technologies locally
---

# Stack

<!-- Keep grounding sparse here. For a concrete wrapper or adapter mention, use:
```markdown
[`someFunction()`](mex://function:<tier-1-id>)
```
-->

## Core Technologies
- **Go** -- service language. [TO BE DETERMINED: Go version]
- **PostgreSQL** -- primary database. [TO BE DETERMINED: version]
- **Redpanda** -- Kafka-API-compatible broker
- **Temporal** -- durable workflow engine, self-hosted dev server
- **Docker Compose** -- local infra for Phase 1
- **Kubernetes on EKS** -- deploy target in later phases

## Key Libraries
- **Temporal Go SDK** (`go.temporal.io/sdk`) -- workflows and activities
- [TO BE DETERMINED: Postgres driver (e.g. pgx) and query layer]
- [TO BE DETERMINED: Kafka client (e.g. franz-go)]
- [TO BE DETERMINED: migration tool]
- [TO BE DETERMINED: HTTP router]

## What We Deliberately Do NOT Use
- No floating-point money types; integer minor units only
- No dual writes (DB + Kafka in the handler); the transactional outbox replaces them
- No managed cloud services in Phase 1; everything runs on Docker Compose at $0

## Version Constraints
[TO BE DETERMINED: populate after first implementation]
