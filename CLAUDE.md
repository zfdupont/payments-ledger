# payments-ledger

## Overview
A Go payments ledger service: double-entry bookkeeping (accounts, balanced journal entries, transfers) over Postgres, with idempotent write APIs, a transactional outbox publishing events to Redpanda (Kafka API), and Temporal workflows for retries and reconciliation. One app built in phases so later infra work (Kubernetes, EKS) deploys a real service.

### Phase 1: Ledger core (current)
- Double-entry ledger in Postgres: accounts, balanced journal entries, transfers.
- Idempotency keys on every write endpoint.
- Transactional outbox publishing events to Redpanda.
- Temporal (self-hosted dev server) for retries and reconciliation workflows.
- Webhook/ingestion from a rate-limited external API, with retries and deduplication.
- OAuth2 client credentials plus API keys.
- Versioned schema migrations, plus a short write-up on tuning one slow query with `EXPLAIN`.
- Runs entirely on Docker Compose locally. No cloud spend in this phase.

### Later phases
Kubernetes deployment on EKS. Scope to be defined per phase.

## Tech Stack
- Language: Go
- Database: PostgreSQL
- Streaming: Redpanda (Kafka API)
- Workflows: Temporal (self-hosted dev server)
- Local infra: Docker Compose
- Deploy target (later phases): Kubernetes on EKS

## Commands
No code yet. Planned commands, to be confirmed once the module exists:
- `docker compose up -d` -- Start Postgres, Redpanda, Temporal locally
- `go run ./cmd/...` -- Run the service
- `go build ./...` -- Build
- `go test ./...` -- Run tests
- `go vet ./...` and `golangci-lint run` -- Lint

## Architecture
Planned flow: HTTP API (OAuth2 client credentials or API key) -> idempotency check -> single Postgres transaction that writes balanced journal entries and an outbox row -> outbox relay publishes to Redpanda. Temporal workflows drive retries and reconciliation. An ingestion worker pulls from or receives webhooks from a rate-limited external API, deduplicates, and posts through the same ledger write path.

Hard rules:
- Every journal entry balances (sum of debits equals sum of credits) and is enforced in the database, not only in Go.
- Ledger rows are append-only. Corrections are reversing entries, never updates or deletes.
- Every write endpoint requires an idempotency key; replays return the original result.
- Events are published only via the outbox, written in the same transaction as the ledger change. Never publish to Kafka directly from a request handler.
- Money is integer minor units plus currency code. No floats.
- Schema changes only through versioned migrations.

Project memory lives in `.mex/` (mex scaffold). Software factory config lives in `factory/`.

## Environment Variables
See `.env.example` for placeholders.
- `DATABASE_URL`
- `KAFKA_BROKERS`, `OUTBOX_TOPIC`
- `TEMPORAL_HOST`, `TEMPORAL_NAMESPACE`, `TEMPORAL_TASK_QUEUE`
- `OAUTH_ISSUER_URL`, `OAUTH_AUDIENCE`, `API_KEY_PEPPER`
- `INGEST_API_BASE_URL`, `INGEST_API_TOKEN`, `INGEST_RATE_LIMIT_RPS`
- `HTTP_ADDR`

## Workflow
- After completing any job (a task, feature, fix, or factory job), invoke the `using-mex-scaffolds` skill to update `.mex/` so project memory matches the code.
