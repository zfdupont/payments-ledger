ACCEPT
Merge as built: the go.mod toolchain change that triggered the escalation is the change the owner asked for in feature.md, it is as small as it can be (go 1.27, no toolchain line, no require lines), and no hard rule is touched.

## Conflict
The feature asks for a new `go.mod`. Any `go.mod` sets the build toolchain through its `go` directive, and that matches the `escalate` entry "adds a dependency or changes the build toolchain" (path `go.mod`). spec.md's Design section says this job escalates by design. The approver found no defects and escalated only because of that rule. So the question is whether the toolchain change needs a separate owner sign-off when the owner's own backlog item asks for it, or whether a different `go` directive would be better.

## Options
### A. Accept as built (chosen)
- Changes: none to code. AC5 in the spec gets a small wording fix so it checks the job's copy of factory.yaml instead of the main checkout, which only changes at merge.
- Effect on users and the backlog: every later job gets real `build`, `test` and `lint` gates. The compose, migration and idempotency jobs and the CI job all need this module. The CI job can pin its Go version to `go.mod` with `go-version-file`.
- Risk against the rules and escalate list: the five Non-Negotiables are about money, ledger tables, idempotency and the outbox, and none of that code exists yet. The escalate entry exists so the owner sees toolchain and dependency changes. Here the owner asked for this one by name, the spec chose and explained the directive (it matches the installed Go 1.27.4), and the result has no `toolchain` line, no `require` lines and no `go.sum`. Later changes to `go.mod` will still escalate, for example pgx, the Temporal SDK or a Kafka client. Accepting this one does not weaken that rule.
- Cost: none.

### B. Lower the `go` directive (for example `go 1.26` or `go 1.25`) for wider compatibility
- Changes: one line in go.mod and AC1, plus another builder round.
- Effect: the only gain would be letting older local Go installs build without downloading a toolchain. The project has one developer, Go 1.27.4 is installed, and CI does not exist yet. The CI job will read the version from go.mod either way. A lower floor would also add a second version to keep in sync with the dev machine.
- Risk: none against the rules, but it is still a `go.mod` change and would escalate again.
- Cost: one rework round for no real benefit. Rejected.

### C. Send to the owner for explicit toolchain sign-off
- Changes: none. The job waits for the owner.
- Effect: delays the job that unblocks the rest of the Foundations backlog, to ask a question the owner already answered by filing this item.
- Why rejected: OWNER is for weakening a hard rule, changing scope or direction, or a real tie. None of those applies. No hard rule is weakened, the scope is exactly what feature.md asks for, and A is clearly better than B.

## Checked
- Read feature.md, spec.md, build.md, triage.md, decision.md, all five round-1 reviews (all PASS, no findings waved away), factory.yaml, backlog.md, and the Non-Negotiables in `.mex/AGENTS.md`.
- `git diff main...HEAD` in the worktree (HEAD 2fa91c3) touches 4 files: go.mod (3 lines: module path plus `go 1.27`), cmd/ledger/main.go, cmd/ledger/main_test.go, and factory/factory.yaml (only `tests` and `commands`; `merge_gate: []` unchanged).
- `go version` reports go1.27.4 darwin/arm64, which matches the `go 1.27` directive.
- I reran the commands in the worktree. `go build ./...` exits 0, `go vet ./...` exits 0, and `go test -count=1 ./...` gives `ok github.com/zfdupont/payments-ledger/cmd/ledger`. `go run ./cmd/ledger | od -c` prints exactly `payments-ledger\n`. `ls go.sum` reports the file as absent.
- Validator: `factory_config.py validate <worktree>` prints `ok` and exits 0. `get <worktree> commands` returns `{"build": "go build ./...", "test": "go test ./...", "lint": "go vet ./..."}`, and `get <worktree> tests` returns `"thin"`. `validate /Users/zfdupont/payments-ledger` also prints `ok`, but only for the old config (`commands: {}`) until merge. That is why AC5 is reworded below.
- Not checked: whether `go 1.27` would cause trouble on another machine or CI runner. There is none yet. The CI backlog item covers it.

## Next step
Spec amendment (replace AC5 in spec.md):

- [x] AC5: `uv run --script /Users/zfdupont/.claude/plugins/cache/zfdupont-plugins/factory/0.3.0/skills/run/scripts/factory_config.py validate <repo>` prints `ok` and exits 0, and `get <repo> commands` returns `{"build": "go build ./...", "test": "go test ./...", "lint": "go vet ./..."}`, where `<repo>` is the job worktree before merge and `/Users/zfdupont/payments-ledger` after merge (test: run the validator)

Leave the rest of spec.md as is. Its Design note that this job escalates by design was correct, and this resolution settles that escalation.

After merge, the post-job mex update must fix what the maintainability review flagged as stale:
- `.mex/AGENTS.md:20` ("No code yet; planned") and the CLAUDE.md Commands section: replace with the three real commands. `golangci-lint run` stays planned.
- `.mex/context/conventions.md:49,55`: fill the Go package layout and directory placeholders with `cmd/ledger`.

Backlog follow-up:
- [ ] Remove the duplicate escalate entry in `factory/factory.yaml`. The free-text line "adds a dependency, changes the build toolchain, or deletes tests" repeats the path-based `go.mod`/`go.sum` entry for dependencies and the toolchain. Keep the path entry and narrow the free-text line to "deletes tests".
