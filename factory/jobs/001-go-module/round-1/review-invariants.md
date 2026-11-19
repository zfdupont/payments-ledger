VERDICT: PASS

No ledger invariant is touched by this diff (HEAD 2fa91c3). I checked each item:

1. Balanced entries: no journal, account or schema code is added, so there is nothing to enforce.
2. Append-only: there are no ledger tables and no UPDATE or DELETE statements.
3. Idempotency: there are no write endpoints. `cmd/ledger/main.go` only prints `payments-ledger` to a writer.
4. Outbox: there is no event code, no Kafka import and no handler.
5. Money: there are no amounts, floats or currency types.
6. Migrations: none were added or edited, and the diff has no `migrations/` path.

The diff has four files: `go.mod`, `cmd/ledger/main.go`, `cmd/ledger/main_test.go` and `factory/factory.yaml`. `go.mod` has no `require` block, and no `go.sum` was added. `factory.yaml` changes only `tests` and `commands`. `/Users/zfdupont/payments-ledger/factory/jobs/001-go-module/round-1/commands.md` records build, test and vet all exiting 0 at HEAD 2fa91c3, which matches the worktree HEAD.
