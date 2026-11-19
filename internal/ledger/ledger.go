// Package ledger is the only non-test code that writes the ledger tables.
package ledger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Line is one debit or credit line. Money is int64 minor units plus a currency code.
type Line struct {
	AccountID   int64
	Direction   string
	AmountMinor int64
	Currency    string
}

// Entry is a journal entry with its lines.
type Entry struct {
	Description string
	Lines       []Line
}

// PostEntry inserts e and its lines in tx and returns the new entry id. It
// does not commit; the balance triggers from 0001 run at the caller's COMMIT.
// There is deliberately no Go-side balance check: the database is the authority.
func PostEntry(ctx context.Context, tx pgx.Tx, e Entry) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		"INSERT INTO journal_entries (description) VALUES ($1) RETURNING id",
		e.Description).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert journal entry: %w", err)
	}
	for _, l := range e.Lines {
		_, err := tx.Exec(ctx,
			`INSERT INTO journal_lines (entry_id, account_id, direction, amount_minor, currency)
			 VALUES ($1, $2, $3, $4, $5)`,
			id, l.AccountID, l.Direction, l.AmountMinor, l.Currency)
		if err != nil {
			return 0, fmt.Errorf("insert journal line: %w", err)
		}
	}
	return id, nil
}
