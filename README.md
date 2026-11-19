# payments-ledger

A double-entry payments ledger service in Go over Postgres, with idempotent writes, a transactional outbox to Redpanda (Kafka API), and Temporal workflows for retries and reconciliation. Built in phases; later phases deploy it to Kubernetes on EKS.

## Status

Phase 1 (ledger core) not started. See `CLAUDE.md` for scope and conventions.

## Local development

Everything runs locally with Docker Compose (Postgres, Redpanda, Temporal dev server). Copy `.env.example` to `.env` and adjust values. The variable list lives in `CLAUDE.md`.
