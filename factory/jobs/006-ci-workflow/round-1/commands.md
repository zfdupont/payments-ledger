# Commands

HEAD: 362354b033806281602390aec952a19791cec60e

## build

`go build -o /dev/null ./...` exit 0

```

```

## test (DATABASE_URL set to compose Postgres)

`DATABASE_URL='postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable' go test -count=1 ./...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.270s
ok  	github.com/zfdupont/payments-ledger/internal/api	1.719s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	1.511s
```

## test (no DATABASE_URL)

`env -u DATABASE_URL go test -count=1 ./...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.274s
ok  	github.com/zfdupont/payments-ledger/internal/api	0.321s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.585s
```

## vet

`go vet ./...` exit 0

```

```

## lint

`go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run` exit 0

```
0 issues.
```

## actionlint

`go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/ci.yml` exit 0

```

```

