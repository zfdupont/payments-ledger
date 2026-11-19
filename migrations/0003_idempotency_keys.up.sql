-- 0003_idempotency_keys: idempotency records for write endpoints.
--
-- One row per (client_id, idempotency_key). The row is inserted in the same
-- transaction as the write it protects, already complete, and is never
-- updated: request_hash identifies the request, and response_status and
-- response_body hold the exact response to replay. response_body is bytea
-- (not jsonb) so a replay returns the stored bytes unchanged. Only 2xx
-- responses are stored. The primary key is what makes concurrent requests
-- with the same key race-free.
--
-- Not a ledger table: it has no append-only triggers.
-- No BEGIN/COMMIT and no schema qualification: the caller owns the
-- transaction and the search_path.

CREATE TABLE idempotency_keys (
    client_id       text        NOT NULL,
    idempotency_key text        NOT NULL CHECK (octet_length(idempotency_key) BETWEEN 1 AND 255),
    request_hash    bytea       NOT NULL CHECK (octet_length(request_hash) = 32),
    response_status integer     NOT NULL CHECK (response_status BETWEEN 200 AND 299),
    response_body   bytea       NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (client_id, idempotency_key)
);
