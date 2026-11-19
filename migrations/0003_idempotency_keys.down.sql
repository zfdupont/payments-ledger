-- 0003_idempotency_keys down: drop the idempotency records.
--
-- idempotency_keys is not a ledger table, so dropping it does not touch the
-- append-only ledger rows.

DROP TABLE idempotency_keys;
