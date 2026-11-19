# Create the Go module and wire factory commands

## Goal
Bootstrap the repo as a Go module with a placeholder `ledger` binary and one smoke test, so the factory has real `build`, `test` and `lint` commands to run on every later job. The repo has no test suite yet; this job adds the first test, and `go build ./...`, `go test ./...` and `go vet ./...` all exiting 0 is the done condition. No ledger hard rule is touched: there is no money, schema, endpoint or event code in this change.

## Behavior
- `go run ./cmd/ledger` from the repo root prints exactly one line and exits 0:
  ```
  payments-ledger
  ```
- `go build ./...`, `go test ./...` and `go vet ./...` each exit 0 from the repo root. `go test ./...` reports `ok  github.com/zfdupont/payments-ledger/cmd/ledger`.
- The factory now runs these three commands for every job: `uv run --script <factory plugin>/skills/run/scripts/factory_config.py validate /Users/zfdupont/payments-ledger` prints `ok`, and `get <repo> commands` returns `{"build": "go build ./...", "test": "go test ./...", "lint": "go vet ./..."}`.

## Design
- `go.mod` (new, repo root):
  - `module github.com/zfdupont/payments-ledger` (matches the owner's GitHub namespace `git@github.com:zfdupont/<repo>.git`; the repo has no remote yet).
  - `go 1.27` directive, matching the installed toolchain (Go 1.27.4). No `toolchain` line, no `require` lines.
  - Standard library only, so no `go.sum` is created.
- `cmd/ledger/main.go` (new), `package main`:
  - `func run(w io.Writer) error` writes `payments-ledger\n` to `w`.
  - `func main()` calls `run(os.Stdout)`; on error it prints the error to stderr and exits 1.
  - No env reading, no network, no HTTP server; those come with later jobs.
- `cmd/ledger/main_test.go` (new), `package main`, standard `testing` package only:
  - `TestRun` calls `run` with a `bytes.Buffer`, asserts a nil error and that the buffer equals `"payments-ledger\n"`.
- `factory/factory.yaml` (edit, only these two keys):
  - `tests: none` becomes `tests: thin` (per the factory schema, `thin` means the test command runs but few tests exercise the code, which is true with one smoke test; `ok` would overstate it).
  - `commands: {}` becomes:
    ```yaml
    commands:
      build: go build ./...
      test: go test ./...
      lint: go vet ./...
    ```
  - Every other key, including `merge_gate: []`, stays as is.
- `golangci-lint` is not installed, so `lint` is `go vet ./...` only. The backlog's CI job adds `golangci-lint run` later.
- Escalation note: the diff touches `go.mod`, which matches the `escalate` path entry for build toolchain changes, so the approver escalates this job by design.

## Acceptance criteria
- [ ] AC1: `go.mod` at the repo root declares `module github.com/zfdupont/payments-ledger` and `go 1.27`, with no `require` block, and no `go.sum` exists (test: read `go.mod`; `go build ./...`)
- [ ] AC2: `run` writes exactly `payments-ledger\n` to the given writer and returns nil (test: `cmd/ledger/main_test.go`)
- [ ] AC3: `go run ./cmd/ledger` prints `payments-ledger` and exits 0 (test: run the program)
- [ ] AC4: `factory/factory.yaml` has `tests: thin` and `commands` with exactly `build: go build ./...`, `test: go test ./...`, `lint: go vet ./...`; no other key changed (test: `git diff factory/factory.yaml`)
- [ ] AC5: `uv run --script /Users/zfdupont/.claude/plugins/cache/zfdupont-plugins/factory/0.3.0/skills/run/scripts/factory_config.py validate /Users/zfdupont/payments-ledger` prints `ok` and exits 0 (test: run the validator)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Out of scope
- `golangci-lint` config or command (later CI job in the backlog).
- Setting `merge_gate` to `[test]`; it stays `[]` unless the owner changes it.
- Any third-party dependency (pgx, Temporal SDK, Kafka client, router).
- HTTP server, config/env loading, internal packages, Docker Compose, migrations.
- Updating `.mex/` scaffold, `CLAUDE.md` commands section or README status (handled by the post-job mex update, not the builder).
