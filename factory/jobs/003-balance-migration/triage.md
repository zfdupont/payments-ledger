LANE: full

Files:
- /Users/zfdupont/payments-ledger/migrations/ (new versioned migration creating accounts, journal_entries, journal_lines and the deferred balance trigger)
- /Users/zfdupont/payments-ledger/internal/ledger/balance_test.go (new Go test against Postgres)

Reasoning: the request adds a new schema migration with a database trigger and a new Go test that exercises it, which is neither docs-only, test-only, comment-only nor config-only, and the migration SQL and new files fall outside the trivial paths `*.md` and `.mex/**`, so it goes to the full lane.
