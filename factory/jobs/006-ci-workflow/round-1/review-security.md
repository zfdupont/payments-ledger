VERDICT: PASS

Scope: the diff changes only `.github/workflows/ci.yml`, `.golangci.yml` and `factory/factory.yaml`. No Go source, auth middleware, migration, token validation, API key hashing, idempotency or SQL code is touched, so the endpoint, token, API key, idempotency and SQL checks have nothing to flag in this diff.

Checked and clean:
1. `.github/workflows/ci.yml`: the trigger is plain `push`. There is no `pull_request_target` or `workflow_run`, so untrusted fork code never runs with secrets. Top-level `permissions: contents: read` gives the least privilege. Checkout uses `persist-credentials: false`. The workflow has no `secrets.*` reference.
2. `.github/workflows/ci.yml:23-27,40`: the Postgres credentials (`ledger` / `ledger-ci`) and `DATABASE_URL` are throwaway values for an ephemeral service container on the runner. They are not real secrets, and the workflow comments say so. They differ from the compose password. The URL uses `sslmode=disable` on localhost only, which is acceptable here.
3. Both actions are pinned to full 40-hex commit SHAs with version comments, so there is no mutable-tag supply-chain risk. The golangci-lint tool is pinned to `@v2.15.0` and run through `go run`, so no third-party action is added. Go module checksum verification applies to `go run pkg@version`.
4. `.github/workflows/ci.yml:49-53`: the Test step interpolates no `${{ }}` expressions into shell, so there is no script-injection surface. `$RUNNER_TEMP` is a runner-controlled value. The test log is written to `RUNNER_TEMP` and holds no secrets. The skip-guard annotation prints no credentials.
5. `.golangci.yml`: the three errcheck exclusions are `Tx.Rollback`, `Conn.Close` and `ResponseWriter.Write`. They do not suppress any security linter, and `gosec` is out of scope per the spec. No `//nolint` was added.
6. `.env` is still gitignored (`.gitignore:2`). The only tracked env file is `.env.example`. No tokens, pepper or keys appear in the diff.
7. `factory/factory.yaml`: only the `commands` block changed.

Non-blocking note (no change required): `round-1/commands.md` records the actionlint run as `@latest`, while the spec pins `@v1.7.12`. This has no security impact on the committed files. The build notes say v1.7.12 was used.
