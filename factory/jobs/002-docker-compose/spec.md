# Add Docker Compose for local Postgres, Redpanda and Temporal

## Goal
Add a root `docker-compose.yml` that starts Postgres, Redpanda and the Temporal dev server on the ports and credentials `.env.example` already uses (`DATABASE_URL`, `KAFKA_BROKERS`, `TEMPORAL_HOST`), each with a healthcheck. The job is done when `docker compose up -d --wait` exits 0 and `docker compose ps` shows all three services healthy. The base has no Go test suite and `commands` is `{}`, so every criterion is checked by running `docker compose` or by reading the file. No ledger hard rule is touched: there is no schema, money, endpoint or event code in this change. The diff adds `docker-compose.yml`, which is on the `escalate` path list (build toolchain), so this job escalates by design.

## Behavior
From the repo root, with or without a `.env` file present:

- `docker compose up -d --wait` pulls the images on first run, starts three containers and exits 0 once all three healthchecks pass (about 30 s on a warm cache).
- `docker compose ps --format '{{.Service}} {{.State}} {{.Health}}'` prints these three lines in any order:
  ```
  postgres running healthy
  redpanda running healthy
  temporal running healthy
  ```
- Postgres accepts the exact `DATABASE_URL` from `.env.example` (`postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable`).
- Redpanda serves the Kafka API at `localhost:9092` (`KAFKA_BROKERS`) and advertises the broker as `localhost:9092`, so a client on the host connects without extra config.
- Temporal's frontend gRPC listens on `localhost:7233` (`TEMPORAL_HOST`), the `default` namespace (`TEMPORAL_NAMESPACE`) exists, and the Web UI is at `http://localhost:8233`.
- All published ports bind to `127.0.0.1` only, so the dev password is not reachable from the LAN.
- `docker compose down` stops everything and keeps Postgres data in the `pgdata` volume. `docker compose down -v` also removes the volume. Temporal dev server state is in memory and resets on restart, which is expected for local dev.

## Design
- `docker-compose.yml` (new, repo root). Top-level `name: payments-ledger`, so every checkout and worktree uses the same project name. The spec writer ran this exact shape (with 0.0.0.0 port binds) on Docker 29.8.0 / Compose v5.5.1: all three went healthy in about 30 s, and AC4 to AC6 passed. Keep it unless a check fails.
  - `postgres`:
    - `image: postgres:18.7-alpine`
    - `environment`: `POSTGRES_USER: ${POSTGRES_USER:-ledger}`, `POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}`, `POSTGRES_DB: ${POSTGRES_DB:-ledger}`. The defaults are the same placeholders `.env.example` already commits. No other literal credential appears in the file.
    - `ports: ["127.0.0.1:5432:5432"]`
    - `volumes: [pgdata:/var/lib/postgresql]`. Postgres 18 images declare the volume at `/var/lib/postgresql` (PGDATA is `/var/lib/postgresql/18/docker`). Mounting at `/var/lib/postgresql/data` as older guides show is wrong for 18.
    - healthcheck: `["CMD-SHELL", "pg_isready -h 127.0.0.1 -U \"$$POSTGRES_USER\" -d \"$$POSTGRES_DB\""]`. `-h 127.0.0.1` matters: the init-time temporary server listens only on the Unix socket, so a socket check would report ready before init finishes.
  - `redpanda`:
    - `image: redpandadata/redpanda:v26.2.7`
    - `command`: `redpanda start --mode=dev-container --smp=1 --memory=1G --kafka-addr=PLAINTEXT://0.0.0.0:9092 --advertise-kafka-addr=PLAINTEXT://localhost:9092` (as a YAML list, one flag per item)
    - `ports: ["127.0.0.1:9092:9092"]`. The admin API, schema registry and HTTP proxy ports are not published.
    - healthcheck: `["CMD-SHELL", "rpk cluster health | grep -E 'Healthy:.+true' || exit 1"]`
  - `temporal`:
    - `image: temporalio/temporal:1.10.1` (the Temporal CLI image; its entrypoint is `temporal`)
    - `command: ["server", "start-dev", "--ip", "0.0.0.0"]`. `--ip 0.0.0.0` is required, or the server listens only on the container's loopback and the published ports do nothing.
    - `ports: ["127.0.0.1:7233:7233", "127.0.0.1:8233:8233"]`
    - healthcheck: `["CMD", "temporal", "operator", "cluster", "health", "--address", "127.0.0.1:7233"]`
  - Every healthcheck uses `interval: 5s`, `timeout: 5s`, `retries: 20`.
  - Top-level `volumes: { pgdata: {} }`.
  - Images are pinned to exact versions. No `latest` tags, no `build:`, no `depends_on` (the services are independent), no `container_name` (it would block parallel projects and adds nothing here).
- `.env.example` (edit). Under `# Postgres`, after `DATABASE_URL`, add these lines and change nothing else:
  ```
  # Used by docker-compose.yml; must match the user, password and db in DATABASE_URL
  POSTGRES_USER=ledger
  POSTGRES_PASSWORD=changeme
  POSTGRES_DB=ledger
  ```
- Do not create, read or edit `.env`. It is gitignored and is not in the job worktree. The compose defaults make the stack start without it.
- Cleanup: after verifying, run `docker compose down -v` so the job leaves no containers, networks or volumes behind.

## Acceptance criteria
- [ ] AC1: `docker compose config --quiet` exits 0 from the repo root with no `.env` file present (test: run `docker compose config --quiet`)
- [ ] AC2: `docker compose up -d --wait` exits 0 from the repo root (test: run it)
- [ ] AC3: `docker compose ps --format '{{.Service}} {{.State}} {{.Health}}'` prints exactly three lines: `postgres running healthy`, `redpanda running healthy`, `temporal running healthy` (test: run it)
- [ ] AC4: the `.env.example` `DATABASE_URL` credentials work: `docker compose exec -T postgres psql "postgres://ledger:changeme@127.0.0.1:5432/ledger?sslmode=disable" -tAc 'select current_user, current_database()'` prints `ledger|ledger` (test: run it)
- [ ] AC5: `docker compose exec -T redpanda rpk cluster info -X brokers=localhost:9092` lists one broker with host `localhost` and port `9092` (test: run it)
- [ ] AC6: `docker compose exec -T temporal temporal operator cluster health --address 127.0.0.1:7233` prints `SERVING`, and `docker compose exec -T temporal temporal operator namespace describe -n default --address 127.0.0.1:7233` exits 0 (test: run both)
- [ ] AC7: `docker compose port postgres 5432`, `docker compose port redpanda 9092`, `docker compose port temporal 7233` and `docker compose port temporal 8233` print `127.0.0.1:5432`, `127.0.0.1:9092`, `127.0.0.1:7233` and `127.0.0.1:8233`, and `curl -s -o /dev/null -w '%{http_code}' http://localhost:8233/` prints `200` (test: run them)
- [ ] AC8: every `image:` in `docker-compose.yml` has an exact version tag (`postgres:18.7-alpine`, `redpandadata/redpanda:v26.2.7`, `temporalio/temporal:1.10.1`), and the only credential literals are the `${VAR:-default}` defaults that match `.env.example` (test: read `docker-compose.yml`)
- [ ] AC9: `.env.example` gains exactly the four lines above (one comment line plus `POSTGRES_USER=ledger`, `POSTGRES_PASSWORD=changeme`, `POSTGRES_DB=ledger`) and no other line changes (test: `git diff .env.example`)
- [ ] AC10: `docker compose down -v` exits 0, and afterwards `docker compose ps -a -q` prints nothing (test: run them)
- [ ] No factory `commands` are configured on base (`commands: {}`), so there are no command gates for this job

## Out of scope
- A Compose service for the ledger app itself, and an internal Kafka listener for in-network clients (add these when the service is containerized).
- Persisting Temporal state (`--db-filename` plus a volume), and publishing the Redpanda admin, schema registry or HTTP proxy ports, or Redpanda Console.
- Creating the `OUTBOX_TOPIC` topic. The outbox relay job owns that.
- An OAuth2 issuer container for `OAUTH_ISSUER_URL` (`localhost:8081`). The auth job owns that.
- Migrations, Go code, and any change to `factory/factory.yaml` (for example, a compose-based test command).
- Updating `.mex/`, the `CLAUDE.md` Commands section or the README. The post-job mex update handles those.
