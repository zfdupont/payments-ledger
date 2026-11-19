VERDICT: CHANGES

Scope: `git diff main...HEAD` at a71c2c0 (docker-compose.yml new, .env.example +4 lines). Commands result (`docker compose config -q`) is exit 0 at the same sha. No patterns in `.mex/patterns/INDEX.md` cover compose or infra (they cover idempotent write endpoints, migrations and event-delivery debugging), so there is no runbook to deviate from. The diff has no ledger writes and no table or column naming.

1. CLAUDE.md:48-55 (Environment Variables): `.env.example` gains three new variables (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `.env.example:4-6`), but the CLAUDE.md Environment Variables list does not name them. The checklist requires every new env var in both places. The spec's Out of scope excludes only `.mex/`, the CLAUDE.md Commands section and the README, so this section is not covered by that exclusion. The spec's Design simply omits the edit, so this is a spec gap, not a builder deviation. Fix: add one line under the list, for example `- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` (docker-compose.yml only; must match `DATABASE_URL`)`. Touch nothing else in CLAUDE.md.

No other defects:
- No dead code, unused config or speculative abstraction. There is no `depends_on`, `container_name`, `build:` or extra published port. Every comment explains a non-obvious flag (`-h 127.0.0.1`, `--ip 0.0.0.0`).
- The `pgdata` volume is used (mounted at `/var/lib/postgresql`, correct for Postgres 18).
- Images are pinned and credentials are only `${VAR:-default}` placeholders that match `.env.example`.
- The `.env.example` comment states the coupling to `DATABASE_URL`.
- `.mex/` (AGENTS.md Commands, architecture.md) is deferred to the post-job mex update per the spec. That is acceptable and not a defect here, but the post-job update must record that `docker compose up -d --wait` now exists and which ports and services it starts.
