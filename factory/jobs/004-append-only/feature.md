# Feature

Add a migration with BEFORE UPDATE OR DELETE (and TRUNCATE) triggers that raise an exception on `journal_entries`, `journal_lines` and any other ledger table, plus `internal/ledger/append_only_test.go` asserting UPDATE and DELETE on each ledger table fail while INSERT of a reversing entry succeeds.
