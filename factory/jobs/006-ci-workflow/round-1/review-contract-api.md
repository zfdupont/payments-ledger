VERDICT: PASS

Reviewed at HEAD 362354b. The diff against main touches only `.github/workflows/ci.yml`, `.golangci.yml` and `factory/factory.yaml`. It has no Go source, migration, handler, route, request shape or response shape changes.

1. No HTTP API contract is affected. The `Idempotency-Key` handling, status codes, error bodies, minor-unit amounts and the existing endpoint shapes are all unchanged. No new endpoints are added, so no new HTTP contract tests are needed. The spec's Out of scope section and AC10 already state that no Go code changes.
2. The `errcheck` exclusion `(net/http.ResponseWriter).Write` in `/Users/zfdupont/payments-ledger/.golangci.yml:15` makes a failed write to the client unreported in the idempotency response path (`writeRaw`). The spec documents this with a written reason and bounds it to one call target, and it does not change the API contract. This is informational and not a defect.
3. The CI Test step fails the run on any `--- SKIP`, so DB-backed HTTP contract tests, including the idempotency replay and conflict tests, can no longer be silently skipped on CI. This supports the contract. The commands result shows the DB-backed tests passing with `DATABASE_URL` set.

No changes required.
