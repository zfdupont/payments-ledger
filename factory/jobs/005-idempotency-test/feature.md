# Feature

Add `internal/api/idempotency_test.go`, a table-driven test that iterates every registered write route (POST/PUT/PATCH/DELETE) in the HTTP router and asserts (a) a replay with the same Idempotency-Key and body returns the original status and body without a second ledger write, (b) the same key with a different body returns a conflict error (409/422), and (c) a missing key is rejected; fail the test if a write route is absent from the table.
