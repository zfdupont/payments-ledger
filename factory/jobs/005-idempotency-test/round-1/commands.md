# Commands

HEAD: d0923ccc4838e9a7b673d4ad26d67c3e3043f08b

## build

`go build -o /dev/null ./...` exit 0

```

```

## test (DATABASE_URL set to compose Postgres)

`DATABASE_URL='postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable' go test -count=1 -v ./... 2>&1 | grep -E '^(---|ok|FAIL|PASS)|^    --- (FAIL|SKIP)'` exit 0

```
--- PASS: TestRun (0.00s)
PASS
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.342s
--- PASS: TestIdempotencyCoverageHelper (0.00s)
--- PASS: TestWriteRoutesCovered (0.00s)
--- PASS: TestWriteRoutesIdempotent (0.15s)
--- PASS: TestPostJournalEntryCreatesRows (0.07s)
--- PASS: TestPostJournalEntryUnbalanced (0.07s)
--- PASS: TestPostJournalEntryUnknownAccount (0.11s)
--- PASS: TestPostJournalEntryInvalidBody (0.18s)
--- PASS: TestPostJournalEntryKeyTooLong (0.05s)
PASS
ok  	github.com/zfdupont/payments-ledger/internal/api	0.840s
--- PASS: TestAppendOnly (0.09s)
--- PASS: TestReversingEntry (0.12s)
--- PASS: TestBalance (0.18s)
PASS
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.820s
```

## test (no DATABASE_URL)

`env -u DATABASE_URL go test -count=1 ./...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/cmd/ledger	0.189s
ok  	github.com/zfdupont/payments-ledger/internal/api	0.261s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	0.529s
```

## lint

`go vet ./...` exit 0

```

```

## race (DATABASE_URL set)

`DATABASE_URL='postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable' go test -count=1 -race ./internal/...` exit 0

```
ok  	github.com/zfdupont/payments-ledger/internal/api	3.733s
ok  	github.com/zfdupont/payments-ledger/internal/ledger	3.233s
```

