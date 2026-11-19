---
name: setup
description: Dev environment setup and commands. Load when setting up the project for the first time or when environment issues arise.
triggers:
  - "setup"
  - "install"
  - "environment"
  - "getting started"
  - "how do I run"
  - "local development"
edges:
  - target: context/stack.md
    condition: when specific technology versions or library details are needed
  - target: context/architecture.md
    condition: when understanding how components connect during setup
  - target: context/conventions.md
    condition: when writing code after setup
# Ground only setup behavior implemented by specific code symbols.
# Entry shape: { node: "function:<tier-1-id>" } (MEX records the fingerprint)
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ5NGEAW9G41KJ3AV8A9GE
  type: guide
  status: promoted
  revision: 4
  title: setup
  relations:
    - type: related_to
      target: mx_01M7SQ5NGJ7NVKAB5BYQW34E29
      note: when specific technology versions or library details are needed
    - type: related_to
      target: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
      note: when understanding how components connect during setup
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: when writing code after setup
---

# Setup

<!-- Commands and environment facts need no code grounding. For a concrete symbol:
```markdown
[`someFunction()`](mex://function:<tier-1-id>)
```
-->

## Prerequisites
- Go toolchain. [TO BE DETERMINED: version]
- Docker with Docker Compose
- Temporal CLI (optional, for inspecting workflows)

## First-time Setup
[TO BE DETERMINED: confirm after the compose file and Go module exist]
1. Copy `.env.example` to `.env` and fill in values
2. `docker compose up -d` (Postgres, Redpanda, Temporal dev server)
3. Run migrations [TO BE DETERMINED: command]
4. Run the service [TO BE DETERMINED: command]

## Environment Variables
- `DATABASE_URL` (required) -- Postgres connection string
- `KAFKA_BROKERS` (required) -- Redpanda brokers
- `OUTBOX_TOPIC` (required) -- topic the outbox relay publishes to
- `TEMPORAL_HOST`, `TEMPORAL_NAMESPACE`, `TEMPORAL_TASK_QUEUE` (required) -- Temporal connection
- `OAUTH_ISSUER_URL`, `OAUTH_AUDIENCE` (required) -- token validation
- `API_KEY_PEPPER` (required) -- secret mixed into API key hashes
- `INGEST_API_BASE_URL`, `INGEST_API_TOKEN` (required for ingestion) -- external API
- `INGEST_RATE_LIMIT_RPS` (optional) -- client-side rate limit
- `HTTP_ADDR` (optional) -- listen address

## Common Commands
Planned, confirm after first implementation:
- `docker compose up -d` -- start local infra
- `go test ./...` -- run tests
- `go vet ./...` / `golangci-lint run` -- lint
- `go build ./...` -- build

## Common Issues
[TO BE DETERMINED: populate after first implementation]
