VERDICT: PASS

Scope reviewed: `git diff main...HEAD` at a71c2c0 touches only `docker-compose.yml` (new) and `.env.example` (+4 lines). I checked it against each ledger invariant. Commands result (`docker compose config -q`) exit 0 at the same sha.

1. Balanced journal entries enforced in the DB: not applicable. The diff has no schema, migration or Go code. Nothing weakens this rule.
2. Append-only ledger (no UPDATE/DELETE): not applicable. No ledger tables or SQL in the diff.
3. Idempotency on write endpoints: not applicable. No endpoints added or changed.
4. Outbox-only events: not applicable. Redpanda is provisioned as infrastructure only. No handler or code publishes to Kafka, and no topic is created (`OUTBOX_TOPIC` is left to the relay job, as the spec says).
5. Money as int64 minor units plus ISO 4217 code, no floats: not applicable. No money handling in the diff.
6. Schema changes only via new versioned migrations: no migrations are added, edited or bypassed. The Postgres service starts empty, with no init scripts mounted that could create schema outside migrations. The `pgdata` mount at `/var/lib/postgresql` is correct for Postgres 18.

Other checks, no defects:
- The only credential literals are `${VAR:-default}` defaults that match the placeholders already committed in `.env.example`.
- All published ports bind to 127.0.0.1.
- Images are pinned to exact versions.
- `.env.example` gains exactly the four specified lines.

No changes required.
