# Idempotent write routes: router, idempotency layer, POST /journal-entries, and the route-coverage test

## Goal
Add `internal/api/idempotency_test.go`. It is a table-driven test that walks every write route (POST/PUT/PATCH/DELETE) registered in the HTTP router. It checks that a replay returns the original response without a second ledger write, that a reused key with a different body gets 422, and that a missing key gets 400. It fails if any write route has no row in the table. That test means nothing unless the code under test exists, and the repo has none of it yet. So this job also adds the smallest production code that makes it real. That is a stdlib `net/http` router that lists its routes, an idempotency layer backed by a new migration `0003`, and one write route, `POST /journal-entries`. The route posts a balanced journal entry through the ledger service, and the idempotency record goes into the same Postgres transaction.

Hard rules: this job puts "every write endpoint requires an idempotency key; replays return the original result" into code, and the test guards it for every future write route. It weakens no rule. Balance stays enforced by the `0001` triggers at COMMIT. Ledger rows are only inserted. Money is `int64` minor units plus a currency code, and JSON with a fractional amount is rejected. Nothing publishes events. The outbox is out of scope, so this route emits no event yet. The handler must not publish to Kafka or anywhere else. The outbox job will add the outbox row inside the same transaction. The router is not served by any binary (`cmd/ledger` stays the placeholder). Unauthenticated money movement therefore stays unreachable until auth lands. As in jobs 003 and 004, DB tests skip when `DATABASE_URL` is unset or unreachable.

## Behavior
No process serves these routes yet. Everything below is seen through `httptest` against `api.New(db)`.

All JSON responses have `Content-Type: application/json`. Error bodies look like `{"error":"<code>","message":"<text>"}`, written with `encoding/json` (no trailing newline required).

`POST /journal-entries`
- Request body: `{"description":"coffee","lines":[{"account_id":1,"direction":"debit","amount_minor":1000,"currency":"USD"},{"account_id":2,"direction":"credit","amount_minor":1000,"currency":"USD"}]}`. `description` is optional and defaults to `""`.
- Success: `201` with body `{"id":<journal_entries.id>}`. In one transaction this inserts one `journal_entries` row, one `journal_lines` row per line, and one `idempotency_keys` row.
- Body that does not decode (bad JSON, unknown field, trailing data after the object, or a non-integer number such as `"amount_minor":10.5`): `400` `{"error":"invalid_body","message":"request body is not a valid journal entry"}`. Nothing is written.
- Rejected by the database, SQLSTATE class 23, at insert or at COMMIT:
  - SQLSTATE `23514` (balance or row CHECK): `422` `{"error":"constraint_violation","message":"<Postgres error message>"}`. Example: `journal entry 7 is unbalanced in USD: debits 1000 <> credits 999`.
  - Any other class-23 code, such as `23503` for an unknown `account_id`: `422` `{"error":"constraint_violation","message":"request violates a database constraint"}`.
  - The whole transaction rolls back, including the idempotency record. The key is not used up, so the client can retry with the same key and a corrected body.
- Any other error: `500` `{"error":"internal","message":"internal error"}`. No error detail is sent to the client.

Idempotency, applied by the router to every write route:
- `Idempotency-Key` header missing or empty: `400` `{"error":"idempotency_key_required","message":"Idempotency-Key header is required"}`. This is checked before the body is read or a transaction begins.
- Key longer than 255 bytes: `400` `{"error":"idempotency_key_invalid","message":"Idempotency-Key must be at most 255 bytes"}`.
- The request hash is SHA-256 over `METHOD + "\n" + URL path + "\n" + raw body bytes`. The same key with a byte-identical body on the same route is a replay. A different body, a different route, or semantically equal JSON with different whitespace counts as a different request.
- Replay of a request that completed: the stored status and the stored body bytes, byte for byte (for example `201` `{"id":7}` again). No handler runs and no row is written.
- Same key, different request hash: `422` `{"error":"idempotency_key_reused","message":"Idempotency-Key was already used with a different request"}`. Nothing is written, and the stored record is unchanged. A later replay with the original body still returns the original response.
- Only 2xx results are stored. 4xx and 5xx results roll back and leave no record.
- Keys are scoped to a client. Until auth exists, every request uses the client id `anonymous`.

## Design
Use stdlib only; `go.mod` gets no new requirements. Go 1.27 `http.ServeMux` method patterns are the router (this settles the "router library" TBD in `.mex/context/stack.md`). The layer follows `.mex/patterns/add-idempotent-write-endpoint.md`: check-or-insert the record in the same transaction as the write, with a unique constraint and no read-then-write race.

- `migrations/0003_idempotency_keys.up.sql` (new). Plain SQL with no `BEGIN`/`COMMIT` and no schema qualification, plus a header comment in the style of `0001`/`0002`:
  ```sql
  CREATE TABLE idempotency_keys (
      client_id       text        NOT NULL,
      idempotency_key text        NOT NULL CHECK (octet_length(idempotency_key) BETWEEN 1 AND 255),
      request_hash    bytea       NOT NULL CHECK (octet_length(request_hash) = 32),
      response_status integer     NOT NULL CHECK (response_status BETWEEN 200 AND 299),
      response_body   bytea       NOT NULL,
      created_at      timestamptz NOT NULL DEFAULT now(),
      PRIMARY KEY (client_id, idempotency_key)
  );
  ```
  `response_body` is `bytea` (not `jsonb`) so a replay returns exactly the stored bytes. A record is written once, already complete, and never updated.
- `migrations/0003_idempotency_keys.down.sql` (new). `DROP TABLE idempotency_keys;` with a comment noting it is not a ledger table. Do not edit `0001_*` or `0002_*`.
- `internal/ledger/ledger.go` (new, `package ledger`). This is the only non-test code that writes ledger tables.
  ```go
  type Line struct { AccountID int64; Direction string; AmountMinor int64; Currency string }
  type Entry struct { Description string; Lines []Line }
  // PostEntry inserts e and its lines in tx. It does not commit; the
  // balance triggers from 0001 run at the caller's COMMIT.
  func PostEntry(ctx context.Context, tx pgx.Tx, e Entry) (int64, error)
  ```
  Use parameterized INSERTs (`journal_entries (description) ... RETURNING id`, then one `journal_lines` insert per line). Do no Go-side balance check, so the database stays the authority.
- `internal/api/router.go` (new, `package api`):
  - `type TxBeginner interface { Begin(context.Context) (pgx.Tx, error) }`. Both `*pgx.Conn` and `*pgxpool.Pool` satisfy it. Tests pass a `*pgx.Conn`.
  - `type Route struct { Method, Path string }`.
  - `type Router struct` holds the route list and an `*http.ServeMux`. `func New(db TxBeginner) *Router` registers `POST /journal-entries`. `func (r *Router) Routes() []Route` returns a copy of the registered routes. `ServeHTTP` delegates to the mux.
  - All registration goes through one unexported method, for example `addWrite(method, path string, h writeHandler)`. It always wraps `h` with the idempotency layer, appends to the route list, and calls `mux.Handle(method+" "+path, ...)`. There is no way to register a write route that skips the layer. `New(nil)` must not panic, so `Routes()` can be tested without a database.
- `internal/api/idempotency.go` (new):
  - `type writeHandler func(ctx context.Context, tx pgx.Tx, body []byte) (status int, resp any, err error)`.
  - An `*apiError{Status int; Code, Message string}` type (implements `error`) and one `writeError` mapping: `*apiError` maps as given, `*pgconn.PgError` class `23` maps as described under Behavior, anything else maps to 500.
  - Wrapper flow:
    1. Validate the key (400s).
    2. Read the body with `io.ReadAll` and compute the hash.
    3. `db.Begin`, then `defer tx.Rollback`.
    4. `SELECT request_hash, response_status, response_body FROM idempotency_keys WHERE client_id = $1 AND idempotency_key = $2`. If a row exists, return the stored response when the hash matches, else the 422.
    5. Otherwise run `h`. On an error or a non-2xx status, write the mapped error or response and return, which rolls back.
    6. `json.Marshal(resp)` once.
    7. `INSERT ... ON CONFLICT (client_id, idempotency_key) DO NOTHING RETURNING 1`. If no row comes back, a concurrent request with the same key committed first. Roll back, re-run the SELECT outside the rolled-back tx, and return its stored response or the 422.
    8. `tx.Commit`. A commit error goes through `writeError` (this is where the deferred balance check surfaces).
    9. Write the status and the exact marshaled bytes.
  - `const anonymousClientID = "anonymous"`, with a comment that auth will replace it.
- `internal/api/journal_entries.go` (new). Request structs with `json` tags `description`, `lines`, `account_id`, `direction`, `amount_minor` (int64), `currency`. Decode with `json.NewDecoder(bytes.NewReader(body))`, `DisallowUnknownFields()`, and a second `Decode` that must return `io.EOF`. Any decode failure maps to `invalid_body`. Call `ledger.PostEntry(ctx, tx, ...)` and return `201, map[string]int64{"id": id}`. The handler never writes SQL itself.
- `internal/api/db_test.go` (new, test-only). `connect(t) *pgx.Conn` is a copy of `internal/ledger/balance_test.go`'s helper, with the same skip messages (`DATABASE_URL not set; skipping Postgres test` / `Postgres unreachable at DATABASE_URL: <error>`). Like that helper, it creates a throwaway `ledger_test_<hex>` schema, applies every `../../migrations/*.up.sql` in sorted order, seeds accounts `cash` and `revenue`, and drops the schema in cleanup. Also add `ledgerRows(t, conn) int`, the sum of `count(*)` over `accounts`, `journal_entries` and `journal_lines`, and `idempotencyRows(t, conn) int`. Do not change `internal/ledger/*_test.go`.
- `internal/api/idempotency_test.go` (new, the requested test):
  - `writeCases` is a `map[string]writeCase` keyed by `"METHOD PATH"`. A `writeCase` has `body`, `otherBody func(t *testing.T, conn *pgx.Conn) []byte` (bodies built from the seeded account ids looked up by name) and `wantStatus int`. Today it has exactly one entry, `"POST /journal-entries"`: debit `cash` and credit `revenue` 1000 USD, with `otherBody` using 2000, and `wantStatus` 201.
  - `coverage(routes []Route, cases map[string]writeCase) (missing, stale []string)` returns write-method routes (POST, PUT, PATCH, DELETE) without a case, and case keys with no matching write route. Non-write methods are ignored. Both results are sorted.
  - `TestIdempotencyCoverageHelper` (no DB). It feeds a synthetic route list `{POST /a, PUT /b, GET /c}` and cases `{POST /a, DELETE /z}`, and asserts `missing == ["PUT /b"]` and `stale == ["DELETE /z"]`.
  - `TestWriteRoutesCovered` (no DB). It runs `coverage(api.New(nil).Routes(), writeCases)` and calls `t.Errorf` for every missing route (`write route %s has no idempotency test case`) and every stale case (`idempotency test case %s matches no write route`).
  - `TestWriteRoutesIdempotent` (DB). It runs one subtest per write route from `New(conn).Routes()`, failing the subtest if the route has no case. Each subtest uses a fresh random key and drives `httptest` requests through the router:
    - (c) No `Idempotency-Key`: status 400, body exactly the `idempotency_key_required` JSON, and `ledgerRows`/`idempotencyRows` unchanged.
    - First request: status `wantStatus`. Record the status and body bytes. `ledgerRows` increased and `idempotencyRows` increased by 1.
    - (a) Replay with the same key and body: identical status and `bytes.Equal` body. `ledgerRows` and `idempotencyRows` unchanged.
    - (b) Same key with `otherBody`: status 422, body exactly the `idempotency_key_reused` JSON, counts unchanged. A further replay with the original body still returns the original status and body.
- `internal/api/journal_entries_test.go` (new, DB tests, small):
  - A post creates one `journal_entries` row whose id equals the response `id`, and two `journal_lines` rows with the given account, direction, amount and currency.
  - An unbalanced body (debit 1000, credit 999 USD) gets 422 `constraint_violation` with a message matching `^journal entry \d+ is unbalanced in USD: debits 1000 <> credits 999$`. `ledgerRows` and `idempotencyRows` are unchanged, and a retry with the same key and a balanced body returns 201.
  - Bodies with `"amount_minor":10.5`, or with an unknown field, get 400 `invalid_body` and write nothing.
  - A key of 256 bytes gets 400 `idempotency_key_invalid`.
- No changes to `cmd/`, `go.mod`/`go.sum`, `docker-compose.yml`, `.env.example`, `factory/factory.yaml`, or `migrations/0001_*`/`0002_*`. If the builder starts compose Postgres, it runs `docker compose down` afterwards, without `-v`.

## Acceptance criteria
- [ ] AC1: `TestWriteRoutesCovered` passes without a database. `api.New(nil).Routes()` has exactly one write route, `POST /journal-entries`, and `writeCases` covers it with no stale entries (test: `internal/api/idempotency_test.go`)
- [ ] AC2: `TestIdempotencyCoverageHelper` passes. `coverage` reports the uncovered `PUT /b` as missing and `DELETE /z` as stale, and ignores `GET /c` (test: `internal/api/idempotency_test.go`)
- [ ] AC3: with compose Postgres up and `DATABASE_URL=postgres://ledger:changeme@localhost:5432/ledger?sslmode=disable`, `go test -count=1 -v ./internal/api/` passes `TestWriteRoutesIdempotent/POST_/journal-entries`. A missing key gets 400 `idempotency_key_required` with no rows written. A replay with the same key and body returns the same status and byte-identical body with no new ledger or idempotency rows. The same key with a different body gets 422 `idempotency_key_reused` with no rows written, and the original replay still works afterwards (test: `internal/api/idempotency_test.go`)
- [ ] AC4: in the same run, a successful `POST /journal-entries` returns 201 `{"id":N}`, and entry `N` exists with exactly the two posted lines (test: `internal/api/journal_entries_test.go`)
- [ ] AC5: in the same run, an unbalanced entry gets 422 `constraint_violation` with message `journal entry <n> is unbalanced in USD: debits 1000 <> credits 999`. It leaves no ledger or idempotency rows, and the same key then succeeds with a balanced body (test: `internal/api/journal_entries_test.go`)
- [ ] AC6: in the same run, `"amount_minor":10.5` and an unknown field each get 400 `invalid_body`, and a 256-byte key gets 400 `idempotency_key_invalid`, with no rows written (test: `internal/api/journal_entries_test.go`)
- [ ] AC7: with `DATABASE_URL` unset, `go test -count=1 -v ./internal/api/ ./internal/ledger/` exits 0. Every DB test prints `--- SKIP`, and `TestWriteRoutesCovered` and `TestIdempotencyCoverageHelper` print `--- PASS` (test: run it)
- [ ] AC8: `migrations/0003_idempotency_keys.up.sql` creates `idempotency_keys` with `PRIMARY KEY (client_id, idempotency_key)` and the CHECKs listed in Design. `git diff main -- migrations/0001_ledger_core.up.sql migrations/0002_ledger_append_only.up.sql cmd/ go.mod` is empty (test: read the file / run the diff)
- [ ] AC9: the only non-test code that INSERTs into `journal_entries` or `journal_lines` is `internal/ledger/ledger.go`. `internal/api` contains no Kafka or outbox code (test: `grep -rniE "INSERT INTO (journal_|accounts)|kafka|outbox" internal/api --include=*.go --exclude=*_test.go` prints nothing)
- [ ] AC10: after the AC3 run, `docker compose exec -T postgres psql -U ledger -d ledger -tAc "select count(*) from pg_namespace where nspname like 'ledger_test_%'"` prints `0` (test: run it)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Out of scope
- Auth (OAuth2 client credentials, API keys) and real per-client key scoping. All keys use client id `anonymous` until then.
- The transactional outbox table, outbox rows for posted entries, and the relay. The auth and outbox jobs must both land before any binary serves this router.
- Serving the router from `cmd/ledger`, `HTTP_ADDR` handling, graceful shutdown, request size limits, timeouts and logging.
- More routes: accounts, transfers, reads, and reversing-entry endpoints.
- Idempotency record expiry or retention, a 409 for in-flight duplicates (the unique-index wait makes the later request return the stored result), and storing non-2xx results.
- Moving the shared `connect` test helper into a common package, a migration runner, CI, and `.mex/` updates (handled by the post-job mex step).
