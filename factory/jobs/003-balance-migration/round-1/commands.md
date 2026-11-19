# Commands

HEAD: 69a8ab49eb4a175025f7c62be7fd2e7f9a9fa145

## build

`go build -o /dev/null ./...` exit 0

```

```

## test (DATABASE_URL set to compose Postgres)

`DATABASE_URL='postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable' go test -count=1 -v ./...` exit 0

```
=== RUN   TestBalance
=== RUN   TestBalance/balanced
=== RUN   TestBalance/balanced_multi_currency
=== RUN   TestBalance/unbalanced
    balance_test.go:187: journal entry 3 is unbalanced in USD: debits 1000 <> credits 999
=== RUN   TestBalance/one_sided
    balance_test.go:187: journal entry 4 is unbalanced in USD: debits 1000 <> credits 0
=== RUN   TestBalance/balanced_total_not_per_currency
    balance_test.go:187: journal entry 5 is unbalanced in EUR: debits 0 <> credits 1000
=== RUN   TestBalance/no_lines
    balance_test.go:187: journal entry 6 has no lines
--- PASS: TestBalance (0.24s)
    --- PASS: TestBalance/balanced (0.04s)
    --- PASS: TestBalance/balanced_multi_currency (0.02s)
    --- PASS: TestBalance/unbalanced (0.01s)
    --- PASS: TestBalance/one_sided (0.01s)
    --- PASS: TestBalance/balanced_total_not_per_currency (0.01s)
    --- PASS: TestBalance/no_lines (0.01s)
PASS
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.460s
```

## test (no DATABASE_URL)

`env -u DATABASE_URL go test -count=1 ./...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.176s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.231s
```

## lint

`go vet ./...` exit 0

```

```

