VERDICT: PASS

Reviewed HEAD 362354b against main. The diff is three files: `.github/workflows/ci.yml`, `.golangci.yml`, `factory/factory.yaml`. All commands in commands.md pass at this sha (build, test with DB, vet, lint with 0 issues, actionlint).

Ledger invariant checklist:
- Balanced journal entries enforced by the DB: not touched. No Go, migration or schema change.
- No UPDATE or DELETE on ledger tables: not touched.
- Idempotency on write endpoints: not touched. No endpoint changed.
- Outbox-only events: not touched. Nothing publishes to Kafka.
- Integer money plus ISO 4217 currency: not touched. No Go source in the diff.
- Migrations: none added or edited. `git diff main...HEAD --name-only` lists only the three files above.

Notes, none blocking:
1. `.golangci.yml:8-9` excludes errcheck for `(pgx.Tx).Rollback`. This is the standard deferred-rollback idiom. It does not hide a ledger defect, because commit errors are still checked. Rollback only ever runs on error paths or after a commit.
2. `.github/workflows/ci.yml` Test step has a skip guard, so the DB-backed ledger tests (balance constraint, append-only) cannot silently skip in CI. This strengthens enforcement of the DB-level invariants. `defaults.run.shell: bash` supplies pipefail, so a failing `go test` piped to `tee` fails the step.
3. The workflow uses throwaway Postgres credentials and no `secrets.*`. Both actions are pinned to SHAs.
