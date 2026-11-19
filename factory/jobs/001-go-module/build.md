## Round 1
Commits: 2fa91c3 Add Go module, placeholder ledger binary, and factory commands
Changed: go.mod, cmd/ledger/main.go, cmd/ledger/main_test.go, factory/factory.yaml
Acceptance criteria:
- [x] AC1 go.mod declares module github.com/zfdupont/payments-ledger and go 1.27, no require, no go.sum (go build ./... passes; ls go.sum -> absent)
- [x] AC2 run writes "payments-ledger\n" and returns nil (TestRun in cmd/ledger/main_test.go)
- [x] AC3 go run ./cmd/ledger prints one line "payments-ledger", exit 0 (verified with od -c)
- [x] AC4 factory.yaml: tests: thin and the three commands only (git diff shows only those lines)
- [x] AC5 factory_config.py validate prints ok, exit 0; get commands returns the expected JSON (run against the worktree copy, which holds the same factory.yaml change)
- [x] go build ./... passes
- [x] go test ./... passes
- [x] go vet ./... passes
Commands: build -> pass, test -> pass (ok github.com/zfdupont/payments-ledger/cmd/ledger), lint -> pass
Notes: AC5 was validated against the worktree path because the main checkout's factory.yaml is unchanged until merge; the spec's literal path (/Users/zfdupont/payments-ledger) will print ok after merge. .mex/ and CLAUDE.md untouched per spec. The Co-Authored-By trailer from the system reminder was not added, per the job rules and the owner's global CLAUDE.md.
