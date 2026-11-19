# Feature

Add `.github/workflows/ci.yml` that on every push runs `go build ./...`, `go vet ./...`, `golangci-lint run` (with a committed `.golangci.yml`) and `go test ./...` against a Postgres service container, and add `lint: golangci-lint run` to `commands` in `factory/factory.yaml` once the config exists; done when the workflow passes on a pushed branch.
