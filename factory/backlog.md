# Backlog

## Foundations

- [x] Create the Go module (`go.mod` with module path for payments-ledger, a minimal `cmd/ledger/main.go`, and one passing smoke test in `cmd/ledger/main_test.go`), then add `build: go build ./...`, `test: go test ./...` and `lint: go vet ./...` under `commands` in `factory/factory.yaml` and replace `tests: none`; done when all three commands exit 0. (job 001-go-module)
- [x] Add `docker-compose.yml` at the repo root that starts Postgres, Redpanda and the Temporal dev server with ports and credentials matching `.env.example` (`DATABASE_URL`, `KAFKA_BROKERS`, `TEMPORAL_HOST`), with healthchecks; done when `docker compose up -d --wait` exits 0 and `docker compose ps` shows all three healthy. (job 002-docker-compose)
- [x] Add the first versioned migration under `migrations/` creating accounts, journal_entries and journal_lines (`amount_minor` bigint, `currency` char(3)) with a deferred constraint trigger that rejects any journal entry whose debits do not equal credits per currency, plus `internal/ledger/balance_test.go` that inserts an unbalanced entry against Postgres and asserts the commit fails and a balanced entry succeeds. (job 003-balance-migration)
- [x] Add a migration with BEFORE UPDATE OR DELETE (and TRUNCATE) triggers that raise an exception on `journal_entries`, `journal_lines` and any other ledger table, plus `internal/ledger/append_only_test.go` asserting UPDATE and DELETE on each ledger table fail while INSERT of a reversing entry succeeds. (job 004-append-only)
- [x] Add `internal/api/idempotency_test.go`, a table-driven test that iterates every registered write route (POST/PUT/PATCH/DELETE) in the HTTP router and asserts (a) a replay with the same Idempotency-Key and body returns the original status and body without a second ledger write, (b) the same key with a different body returns a conflict error (409/422), and (c) a missing key is rejected; fail the test if a write route is absent from the table. (job 005-idempotency-test)
- [ ] Add `.github/workflows/ci.yml` that on every push runs `go build ./...`, `go vet ./...`, `golangci-lint run` (with a committed `.golangci.yml`) and `go test ./...` against a Postgres service container, and add `lint: golangci-lint run` to `commands` in `factory/factory.yaml` once the config exists; done when the workflow passes on a pushed branch.


## Follow-ups from job 001-go-module

- [ ] Add `/ledger` to `.gitignore`: `go build ./...` in the repo root writes a `ledger` binary there (seen in jobs 001 and its reviews), and it is currently untracked but not ignored.
- [ ] Add a test for the error path of `cmd/ledger/main.go` (`run` returns an error -> message on stderr, exit 1), which the round 1 logic review noted is untested.
- [ ] Set `merge_gate: [test]` in `factory/factory.yaml` so `go test ./...` reruns on `main` after each merge.

## Follow-ups from job 002-docker-compose

- [ ] Before the EKS phase, replace the local-only `docker-compose.yml` defaults (Postgres `changeme` password, unauthenticated Redpanda and Temporal) with authenticated settings for any non-local environment, as noted in the round 1 security review.

## Follow-ups from job 003-balance-migration

- [ ] Pin `search_path` on `check_journal_entry_balanced()` (e.g. `SET search_path = pg_catalog, <schema>`) once a migration runner and target schema are chosen, so the trigger cannot resolve `journal_lines` through a caller-controlled `search_path`.
- [ ] Choose and wire a migration runner (e.g. golang-migrate or goose) with a `migrate up` command documented in `.mex/context/setup.md`; today `migrations/0001_ledger_core.up.sql` is only applied by `internal/ledger/balance_test.go`.
- [ ] Add a strict mode for DB tests (e.g. `LEDGER_REQUIRE_DB=1`) so `internal/ledger` tests fail instead of skipping when Postgres is unreachable, for use in CI.
- [ ] Check that a journal line's `currency` matches its account's currency (trigger or FK on `(account_id, currency)`), with a test in `internal/ledger`.
- [ ] When the HTTP layer lands, map SQLSTATE 23514 from the balance trigger to a stable 422 error with a typed code instead of raw Postgres text, with a test in `internal/api`.

## Follow-ups from job 004-append-only

- [ ] Close privileged bypasses of the append-only triggers: create a non-owner application DB role (no TRIGGER/TRUNCATE privilege, cannot set `session_replication_role`) used by the service, with a test in `internal/ledger` that `ALTER TABLE ... DISABLE TRIGGER` fails for that role.
- [ ] Add a `reverses_entry_id` column (nullable FK to `journal_entries`) via a new migration so a reversing entry links to the entry it corrects, with a test that a reversal must net the original to zero.
- [ ] Shorten the subtest names in `internal/ledger/append_only_test.go` (they embed full SQL text) to `<table>/<op>` labels.

## Follow-ups from job 005-idempotency-test

- [ ] Add a committed test in `internal/api` for concurrent same-key requests (per-request connections from a pool) that asserts exactly one journal entry is written and every same-body request gets the identical stored response (exercises `replayCommitted`).
- [ ] Cap request body size in `internal/api/idempotency.go` (replace bare `io.ReadAll` with `http.MaxBytesReader`) and return 413 with the JSON error shape, with a test.
- [ ] Return 400 (not 500) for an `Idempotency-Key` that is not valid UTF-8, with a test in `internal/api/journal_entries_test.go`.
- [ ] Return JSON error bodies for unmatched routes and wrong methods (404/405) from the `internal/api` router instead of the mux's plain text.
- [ ] Serve the router from `cmd/ledger` (listen on `HTTP_ADDR`, connect via `DATABASE_URL`) only together with auth, so no unauthenticated write route is ever exposed.
- [ ] When auth lands, replace the constant `anonymous` client id in `internal/api` idempotency scoping with the authenticated client id, and add a test that two clients can use the same key independently.
