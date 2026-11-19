## Round 1
Commits: 362354b Add CI workflow, golangci-lint config, and lint command
Changed: .github/workflows/ci.yml, .golangci.yml, factory/factory.yaml
Acceptance criteria:
- [x] AC1 actionlint v1.7.12 on ci.yml exits 0, no output (actionlint command)
- [x] AC2 push trigger unfiltered, contents: read, one ubuntu-24.04 job, bash default shell, postgres:18.7-alpine with pg_isready -h 127.0.0.1, 5432:5432, DATABASE_URL matches service creds, no secrets. (read ci.yml; file is the spec verbatim)
- [x] AC3 step order checkout (persist-credentials false), setup-go (go-version-file), build, vet, lint, test (read ci.yml)
- [x] AC4 uses-pin grep prints nothing (grep exit 1)
- [x] AC5 `golangci-lint config verify` exits 0; .golangci.yml is the spec content with three commented exclude-functions (config verify, read)
- [x] AC6 `run --max-same-issues=0 --max-issues-per-linter=0` prints `0 issues.`, exit 0; `git grep nolint` finds nothing (golangci-lint command)
- [x] AC7 Test step script run locally via `RUNNER_TEMP=$(mktemp -d) bash -eo pipefail -c` against compose Postgres: exit 0, 34 `--- PASS` lines including internal/api and internal/ledger DB tests, no `--- SKIP`
- [x] AC8 Same script with DATABASE_URL unset: exit 1, printed `::error::Postgres tests were skipped; check the postgres service and DATABASE_URL`
- [x] AC9 factory.yaml commands are build/test/vet/lint as specified; version string v2.15.0 identical in both files
- [x] AC10 `git diff --name-only main` lists only the three files (plus job folder, which is untracked on main)
Commands: build -> pass; test -> pass; vet -> pass (go vet ./...); lint -> pass (golangci-lint v2.15.0, 0 issues)
Notes: No code changes were needed; the 12 errcheck findings are all covered by the three exclude-functions. The stray `ledger` binary was removed. Compose stack left running. The commit has no attribution lines, per the global CLAUDE.md and job instructions. The owner check on a real pushed branch remains out of scope.
