# Feature

Add the first versioned migration under `migrations/` creating accounts, journal_entries and journal_lines (`amount_minor` bigint, `currency` char(3)) with a deferred constraint trigger that rejects any journal entry whose debits do not equal credits per currency, plus `internal/ledger/balance_test.go` that inserts an unbalanced entry against Postgres and asserts the commit fails and a balanced entry succeeds.
