# Commands

HEAD: 7a0961d0145b5ac076043be2364c461a657ad0cc

## build

`go build -o /dev/null ./...` exit 0

```

```

## test (DATABASE_URL set to compose Postgres)

`DATABASE_URL='postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable' go test -count=1 -v ./... 2>&1 | grep -E '^(---|ok|FAIL|PASS)' | grep -v '    ---'` exit 0

```
--- PASS: TestRun (0.00s)
PASS
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.179s
--- PASS: TestAppendOnly (0.22s)
--- PASS: TestReversingEntry (0.22s)
--- PASS: TestBalance (0.28s)
PASS
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.943s
```

## test (no DATABASE_URL)

`env -u DATABASE_URL go test -count=1 ./...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.182s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.213s
```

## lint

`go vet ./...` exit 0

```

```

