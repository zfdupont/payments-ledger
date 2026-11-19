LANE: full

Files:
- go.mod
- cmd/ledger/main.go
- cmd/ledger/main_test.go
- factory/factory.yaml

Reasoning: The request adds Go source and a new go.mod, which is code outside the trivial paths (`*.md`, `.mex/**`), and go.mod matches the escalate entry for adding a dependency or changing the build toolchain, so this is full.
