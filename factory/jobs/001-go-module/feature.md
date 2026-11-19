# Feature

Create the Go module (`go.mod` with module path for payments-ledger, a minimal `cmd/ledger/main.go`, and one passing smoke test in `cmd/ledger/main_test.go`), then add `build: go build ./...`, `test: go test ./...` and `lint: go vet ./...` under `commands` in `factory/factory.yaml` and replace `tests: none`; done when all three commands exit 0.
