# CI workflow: build, vet, golangci-lint and Postgres-backed tests on every push

## Goal
Add `.github/workflows/ci.yml`. On every push it runs `go build ./...`, `go vet ./...`, golangci-lint (configured by a new committed `.golangci.yml`) and `go test ./...` against a Postgres service container, and it fails if the DB tests skip instead of running. Also wire golangci-lint into `factory/factory.yaml` so the factory runs the same lint locally. golangci-lint is not installed locally and must not need a global install. The factory and CI both run it through one pinned `go run ...@v2.15.0` invocation.

The feature's done condition is "passes on a pushed branch". This job cannot check it: the repo has no git remote and the factory never pushes. It becomes an owner check after merge (see Out of scope). Inside the job it is replaced by local stand-ins: actionlint on the workflow, `golangci-lint config verify`, and every CI step run locally against the compose Postgres with no skipped tests.

The repo has a Go test suite, but there is no Go-level test for a YAML file. Criteria here are checked with commands (actionlint, golangci-lint, the CI test script run locally) and by reading the files. No hard rule in `.mex/AGENTS.md#Non-Negotiables` is touched: no Go source, migration, schema or ledger behavior changes. The lint config excludes exactly three errcheck call targets, each with a written reason. No linter is disabled.

## Behavior
On GitHub, after the owner adds a remote and pushes:
- Every push to any branch starts one workflow run named `CI`. It has a single job, `build, vet, lint, test`, on `ubuntu-24.04`, with a `postgres:18.7-alpine` service container (the same image as `docker-compose.yml`).
- Steps run in this order: checkout, setup Go (version from `go.mod`), `Build`, `Vet`, `Lint`, `Test`. The first failing step fails the run, and later steps do not run.
- `Lint` prints golangci-lint findings in its default `file:line:col: message (linter)` format and exits non-zero on any finding.
- `Test` runs `go test -count=1 -v ./...`, which applies migrations itself into a throwaway schema per test. If any test reports `--- SKIP`, the step fails with this annotation:
  `::error::Postgres tests were skipped; check the postgres service and DATABASE_URL`
  The only skips in the repo are the DB-unavailable skips in `internal/api/db_test.go` and `internal/ledger/balance_test.go`. This turns a silently green run with no DB coverage into a red one.

Locally (factory and developers):
- `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run` prints `0 issues.` and exits 0 on the current tree. The first run downloads and builds the tool, which takes about a minute.
- `go vet ./...` still runs as its own factory command.

## Design
Files to add or change. No Go code, migrations, `go.mod` or `go.sum` change. `go run pkg@version` does not touch `go.mod`.

### `.golangci.yml` (new, repo root)
Use golangci-lint v2 config format with the `standard` linter set (errcheck, govet, ineffassign, staticcheck, unused). I verified this against the tree with v2.15.0. With the default config there are exactly 12 findings, all errcheck, and all are one of three idioms:
- `defer tx.Rollback(ctx)` after a commit path, plus one explicit rollback before `replayCommitted` (`internal/api/idempotency.go` lines 98, 134, 158; `internal/ledger/append_only_test.go:28`; `internal/ledger/balance_test.go:98`)
- `conn.Close(...)` in test cleanup and skip paths (`internal/api/db_test.go` lines 33, 42, 52; `internal/ledger/balance_test.go` lines 35, 44, 54)
- `w.Write(body)` in `writeRaw` (`internal/api/idempotency.go:68`)

Fix these with a narrow `exclude-functions` list, not with code churn or `//nolint`. Content:

```yaml
version: "2"

linters:
  default: standard
  settings:
    errcheck:
      exclude-functions:
        # Deferred rollback is the pgx idiom: after Commit it returns ErrTxClosed,
        # and on an error path the original error is what gets reported.
        - (github.com/jackc/pgx/v5.Tx).Rollback
        # Closing a connection in test cleanup or skip paths has no caller to report to.
        - (*github.com/jackc/pgx/v5.Conn).Close
        # A failed response write means the client is gone; status and headers are
        # already sent, so a handler has nothing left to do with the error.
        - (net/http.ResponseWriter).Write
```

Do not add `disable:`, `exclusions:` rules or paths, `issues.max-*` caps, or `//nolint` comments. If a different finding appears when the builder runs it, fix it in code. Do not widen this list.

### `.github/workflows/ci.yml` (new)
- Pin both actions to full commit SHAs with a version comment. Both tags are lightweight, so these are commit SHAs (resolved with `git ls-remote`):
  - `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1`
  - `actions/setup-go@d8cbb8045d7f43706b9177cceb36c1f91a6644d5 # v7.1.0`
- Run golangci-lint with the same `go run` command as the factory, not `golangci/golangci-lint-action`. That keeps one pinned version string and one invocation, and adds no third-party action.
- Set `defaults.run.shell: bash`. Only an explicit `bash` gets `-eo pipefail` on GitHub runners, and without pipefail a failing `go test` piped into `tee` would pass.
- `-count=1` stops a test cache restored by setup-go from replaying an old result.
- The Postgres credentials are throwaway values for an ephemeral CI container with no data and no network exposure beyond the runner. They are not secrets, and the workflow uses no `secrets.*`. The health check uses `-h 127.0.0.1` for the same reason given in `docker-compose.yml` (the init-time server listens only on the Unix socket).

```yaml
name: CI

on:
  push:

permissions:
  contents: read

jobs:
  ci:
    name: build, vet, lint, test
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    defaults:
      run:
        shell: bash
    services:
      postgres:
        # Same image as docker-compose.yml. Throwaway credentials for an ephemeral container.
        image: postgres:18.7-alpine
        env:
          POSTGRES_USER: ledger
          POSTGRES_PASSWORD: ledger-ci
          POSTGRES_DB: ledger
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -h 127.0.0.1 -U ledger -d ledger"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 20
    env:
      DATABASE_URL: postgres://ledger:ledger-ci@localhost:5432/ledger?sslmode=disable
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@d8cbb8045d7f43706b9177cceb36c1f91a6644d5 # v7.1.0
        with:
          go-version-file: go.mod
      - name: Build
        run: go build ./...
      - name: Vet
        run: go vet ./...
      - name: Lint
        run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run
      - name: Test
        run: |
          go test -count=1 -v ./... 2>&1 | tee "$RUNNER_TEMP/go-test.log"
          if grep -q -- '--- SKIP' "$RUNNER_TEMP/go-test.log"; then
            echo "::error::Postgres tests were skipped; check the postgres service and DATABASE_URL"
            exit 1
          fi
```

### `factory/factory.yaml` (edit `commands` only)
```yaml
commands:
  build: go build ./...
  test: go test ./...
  vet: go vet ./...
  lint: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run
```
`go vet` moves from `lint` to a new `vet` key so it is still enforced, and `lint` becomes golangci-lint as the feature asks. The command uses the pinned `go run` form because no global binary exists. Change nothing else in the file.

## Acceptance criteria
- [ ] AC1: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/ci.yml` exits 0 with no output. (test: actionlint command)
- [ ] AC2: `.github/workflows/ci.yml` triggers on `push` with no branch or path filter. It sets top-level `permissions: contents: read`, has one job on `ubuntu-24.04` with `defaults.run.shell: bash`, and has a `postgres:18.7-alpine` service with a `pg_isready -h 127.0.0.1` health check and port `5432:5432`. It sets job-level `DATABASE_URL` that matches the service's user, password and db, and contains no `secrets.` reference. (test: read `.github/workflows/ci.yml`)
- [ ] AC3: The workflow steps are checkout (with `persist-credentials: false`), setup-go (with `go-version-file: go.mod`), then `go build ./...`, `go vet ./...`, `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run`, then the Test script above, in that order. (test: read `.github/workflows/ci.yml`)
- [ ] AC4: Every `uses:` line in the workflow pins a 40-hex commit SHA followed by a `# v` version comment. `grep -E 'uses:' .github/workflows/ci.yml | grep -vE '@[0-9a-f]{40} # v[0-9]'` prints nothing. (test: grep command)
- [ ] AC5: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 config verify` exits 0. `.golangci.yml` has `version: "2"` and `linters.default: standard`. Its only customization is the three errcheck `exclude-functions` entries listed above, each with a justification comment. It has no `disable`, `exclusions` or `issues` keys. (test: config verify command and read `.golangci.yml`)
- [ ] AC6: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run --max-same-issues=0 --max-issues-per-linter=0` prints `0 issues.` and exits 0, and `git grep -n 'nolint'` finds nothing. (test: golangci-lint command)
- [ ] AC7: Local CI stand-in. With `docker compose up -d --wait` running and `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable`, running the Test step body with `RUNNER_TEMP=$(mktemp -d) bash -eo pipefail -c '<Test step run script>'` exits 0, and the log contains `--- PASS` lines for the `internal/api` and `internal/ledger` DB tests and no `--- SKIP`. (test: run the step script locally)
- [ ] AC8: The skip guard works. Running the same command with `DATABASE_URL` unset exits non-zero and prints `::error::Postgres tests were skipped; check the postgres service and DATABASE_URL`. (test: run the step script locally)
- [ ] AC9: `factory/factory.yaml` `commands` is exactly `build: go build ./...`, `test: go test ./...`, `vet: go vet ./...`, `lint: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run`. The golangci-lint version string is identical in `factory/factory.yaml` and `.github/workflows/ci.yml`. (test: read both files)
- [ ] AC10: `git diff --stat main` lists only `.github/workflows/ci.yml`, `.golangci.yml` and `factory/factory.yaml` (plus job-folder files). No `*.go`, `migrations/*`, `go.mod` or `go.sum` changes. (test: git diff command)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run` passes (new `lint` command)

## Out of scope
- Owner check after merge (not verifiable in this job): add a GitHub remote, push a branch, and confirm the `CI` run is green with no `--- SKIP` in the `Test` log. That is the feature's "passes on a pushed branch" condition.
- A strict DB-test mode in Go (`LEDGER_REQUIRE_DB=1`, backlog item). The CI skip guard is a workflow-only stand-in for it.
- `pull_request` triggers, a branch-protection merge gate, `merge_gate` entries in `factory.yaml`, `-race`, coverage, and caching beyond setup-go's default.
- Redpanda and Temporal service containers (no test uses them yet).
- Extra linters beyond the `standard` set (gosec, revive, and so on), and code refactors to remove the excluded idioms.
- Updating `.mex/` and the `CLAUDE.md` Commands section. The post-job mex GROW step handles those (for example `.mex/AGENTS.md` still says golangci-lint is "planned, not installed yet").
