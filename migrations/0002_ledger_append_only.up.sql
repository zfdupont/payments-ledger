-- 0002_ledger_append_only: make the ledger tables append-only.
--
-- Adds reject_ledger_mutation() and a statement-level
-- BEFORE UPDATE OR DELETE OR TRUNCATE trigger on accounts, journal_entries
-- and journal_lines. Each trigger raises restrict_violation (23001), so a
-- committed row can never be changed or removed. Corrections are reversing
-- journal entries (new INSERTs), never UPDATE or DELETE.
--
-- The triggers are FOR EACH STATEMENT because TRUNCATE only supports
-- statement-level triggers; it also rejects UPDATE and DELETE that match no
-- rows. INSERT ... ON CONFLICT DO UPDATE is rejected as an UPDATE.
--
-- TRUNCATE on accounts or journal_entries without CASCADE is already refused
-- by Postgres's foreign-key check (0A000) before any trigger fires. Only
-- TRUNCATE ... CASCADE reaches these triggers on FK-referenced tables.
--
-- Privileged bypasses (ALTER TABLE ... DISABLE TRIGGER, session_replication_role,
-- DROP TRIGGER by the owner) are not blocked here; they belong with DB roles.
--
-- No BEGIN/COMMIT and no schema qualification: the caller owns the
-- transaction and the search_path.

CREATE FUNCTION reject_ledger_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only: % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'restrict_violation',
              HINT = 'Ledger rows are never changed; post a reversing journal entry to correct one.';
END;
$$;

CREATE TRIGGER accounts_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON accounts
    FOR EACH STATEMENT EXECUTE FUNCTION reject_ledger_mutation();

CREATE TRIGGER journal_entries_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON journal_entries
    FOR EACH STATEMENT EXECUTE FUNCTION reject_ledger_mutation();

CREATE TRIGGER journal_lines_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON journal_lines
    FOR EACH STATEMENT EXECUTE FUNCTION reject_ledger_mutation();
