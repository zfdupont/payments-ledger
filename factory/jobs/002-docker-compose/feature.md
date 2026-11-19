# Feature

Add `docker-compose.yml` at the repo root that starts Postgres, Redpanda and the Temporal dev server with ports and credentials matching `.env.example` (`DATABASE_URL`, `KAFKA_BROKERS`, `TEMPORAL_HOST`), with healthchecks; done when `docker compose up -d --wait` exits 0 and `docker compose ps` shows all three healthy.
