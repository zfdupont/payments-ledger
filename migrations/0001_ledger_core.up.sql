-- 0001_ledger_core: core double-entry ledger tables.
--
-- Creates accounts, journal_entries and journal_lines, plus the function
-- check_journal_entry_balanced() and two deferred constraint triggers that
-- enforce the balance invariant in Postgres. Balance is checked at COMMIT:
-- for every currency in an entry, debits must equal credits, and an entry
-- must have at least one line. Row-level CHECKs reject bad lines at INSERT.
--
-- Money is bigint minor units plus a three-letter currency code.
-- No BEGIN/COMMIT and no schema qualification: the caller owns the
-- transaction and the search_path.

CREATE TABLE accounts (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE journal_entries (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    description text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE journal_lines (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entry_id     bigint NOT NULL REFERENCES journal_entries (id),
    account_id   bigint NOT NULL REFERENCES accounts (id),
    direction    text NOT NULL CHECK (direction IN ('debit', 'credit')),
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    currency     char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX journal_lines_entry_id_idx ON journal_lines (entry_id);

CREATE FUNCTION check_journal_entry_balanced() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_entry_id bigint;
    v_currency text;
    v_debits   numeric;
    v_credits  numeric;
BEGIN
    IF TG_TABLE_NAME = 'journal_entries' THEN
        v_entry_id := NEW.id;
    ELSE
        v_entry_id := NEW.entry_id;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM journal_lines WHERE entry_id = v_entry_id) THEN
        RAISE EXCEPTION 'journal entry % has no lines', v_entry_id
            USING ERRCODE = 'check_violation';
    END IF;

    SELECT currency,
           coalesce(sum(amount_minor) FILTER (WHERE direction = 'debit'), 0),
           coalesce(sum(amount_minor) FILTER (WHERE direction = 'credit'), 0)
      INTO v_currency, v_debits, v_credits
      FROM journal_lines
     WHERE entry_id = v_entry_id
     GROUP BY currency
    HAVING coalesce(sum(amount_minor) FILTER (WHERE direction = 'debit'), 0)
        <> coalesce(sum(amount_minor) FILTER (WHERE direction = 'credit'), 0)
     ORDER BY currency
     LIMIT 1;

    IF FOUND THEN
        RAISE EXCEPTION 'journal entry % is unbalanced in %: debits % <> credits %',
            v_entry_id, v_currency, v_debits, v_credits
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER journal_entries_balanced
    AFTER INSERT ON journal_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_journal_entry_balanced();

CREATE CONSTRAINT TRIGGER journal_lines_balanced
    AFTER INSERT ON journal_lines
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_journal_entry_balanced();
