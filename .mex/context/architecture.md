---
name: architecture
description: How the major pieces of this project connect and flow. Load when working on system design, integrations, or understanding how components interact.
triggers:
  - "architecture"
  - "system design"
  - "how does X connect to Y"
  - "integration"
  - "flow"
edges:
  - target: context/stack.md
    condition: when specific technology details are needed
  - target: context/decisions.md
    condition: when understanding why the architecture is structured this way
  - target: context/conventions.md
    condition: when implementing a component described here
  - target: patterns/INDEX.md
    condition: when starting a task on one of these components
  - target: patterns/add-idempotent-write-endpoint.md
    condition: when adding a write endpoint on the ledger write path
  - target: patterns/add-migration.md
    condition: when a component needs schema changes
  - target: patterns/debug-event-delivery.md
    condition: when events from the outbox relay or Temporal are missing or duplicated
# Broad overview: keep this empty unless a claim depends on a few specific symbols.
# Entry shape: { node: "function:<tier-1-id>" } (MEX records the fingerprint)
grounds_to: []
last_updated: 2026-11-19
mex:
  id: mx_01M7SQ5NG2T3V9BSHGFF8CDBY1
  type: architecture
  status: promoted
  revision: 4
  title: architecture
  relations:
    - type: related_to
      target: mx_01M7SQ5NGJ7NVKAB5BYQW34E29
      note: when specific technology details are needed
    - type: related_to
      target: mx_01M7SQ8TQ4XGJT42CCJFDZ8QT4
      note: when understanding why the architecture is structured this way
    - type: related_to
      target: mx_01M7SQ5NGAK5NFEJGTJ9MQEYHG
      note: when implementing a component described here
---

# Architecture

<!-- Read broad, ground tight. Architecture usually grounds sparsely. When a
     specific symbol is worth navigating to, use this inline form:
```markdown
[`someFunction()`](mex://function:<tier-1-id>)
```
-->

## System Overview
Planned flow (nothing built yet):

Client -> HTTP API -> auth (OAuth2 client credentials or API key) -> idempotency check
(key + request hash; a replay returns the stored response) -> ledger service opens one
Postgres transaction -> writes balanced journal entry lines + outbox row + idempotency
record -> commit -> outbox relay polls unpublished rows and publishes to Redpanda ->
marks rows published.

Temporal workflows wrap anything that must retry or reconcile: posting ingested external
events, and periodic reconciliation of ledger balances against the external source.
The ingestion worker receives webhooks or pulls from a rate-limited external API,
deduplicates by external event id, and posts through the same ledger write path.

## Key Components
- **HTTP API** -- write and read endpoints; every write requires an `Idempotency-Key`. [TO BE DETERMINED: router library]
- **Ledger service** -- accounts, journal entries, transfers; the only code that writes ledger tables
- **Idempotency store** -- Postgres table keyed by (client, key); stores request hash and response
- **Outbox relay** -- publishes outbox rows to Redpanda in order, at-least-once
- **Temporal worker** -- retry and reconciliation workflows and their activities
- **Ingestion worker** -- rate-limited client for the external API plus webhook receiver, dedup by external id
- **Auth** -- OAuth2 client-credentials token validation and hashed API keys

## External Dependencies
- **PostgreSQL** -- system of record; balance and append-only invariants enforced in schema/constraints
- **Redpanda (Kafka API)** -- event stream fed only by the outbox relay; consumers must tolerate duplicates
- **Temporal (self-hosted dev server)** -- durable retries and reconciliation; activities must be idempotent
- **External rate-limited API** -- ingestion source. [TO BE DETERMINED: which API]
- **OAuth2 issuer** -- issues client-credentials tokens. [TO BE DETERMINED: which issuer for local dev]

## What Does NOT Exist Here
- No direct Kafka publishing from request handlers; only the outbox relay publishes
- No updates or deletes on ledger rows; corrections are reversing entries
- No cloud infra in Phase 1; Kubernetes/EKS comes in a later phase
- No UI; this is an API service
