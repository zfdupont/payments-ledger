package ledger

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// namedLine is a journal line that names its account explicitly.
type namedLine struct {
	account   string
	direction string
	amount    int64
	currency  string
}

// postNamed inserts an entry and its lines in one transaction and returns
// the entry id and the commit error.
func postNamed(ctx context.Context, conn *pgx.Conn, description string, lines []namedLine) (int64, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx,
		"INSERT INTO journal_entries (description) VALUES ($1) RETURNING id", description).Scan(&id); err != nil {
		return 0, err
	}
	for _, l := range lines {
		_, err := tx.Exec(ctx,
			`INSERT INTO journal_lines (entry_id, account_id, direction, amount_minor, currency)
			 VALUES ($1, (SELECT id FROM accounts WHERE name = $2), $3, $4, $5)`,
			id, l.account, l.direction, l.amount, l.currency)
		if err != nil {
			return id, err
		}
	}
	return id, tx.Commit(ctx)
}

var originalLines = []namedLine{
	{"cash", "debit", 1000, "USD"},
	{"revenue", "credit", 1000, "USD"},
}

func TestAppendOnly(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	id, err := postNamed(ctx, conn, "", originalLines)
	if err != nil {
		t.Fatalf("post original entry: %v", err)
	}

	tests := []struct {
		table string
		op    string
		sql   string
		args  []any
	}{
		{"accounts", "UPDATE", "UPDATE accounts SET name = name || '_x' WHERE name = 'cash'", nil},
		{"accounts", "DELETE", "DELETE FROM accounts WHERE name = 'cash'", nil},
		{"accounts", "TRUNCATE", "TRUNCATE accounts CASCADE", nil},
		{"journal_entries", "UPDATE", "UPDATE journal_entries SET description = 'edited' WHERE id = $1", []any{id}},
		{"journal_entries", "DELETE", "DELETE FROM journal_entries WHERE id = $1", []any{id}},
		{"journal_entries", "TRUNCATE", "TRUNCATE journal_entries CASCADE", nil},
		{"journal_lines", "UPDATE", "UPDATE journal_lines SET amount_minor = 1 WHERE entry_id = $1", []any{id}},
		{"journal_lines", "UPDATE", "UPDATE journal_lines SET amount_minor = 1 WHERE false", nil},
		{"journal_lines", "DELETE", "DELETE FROM journal_lines WHERE entry_id = $1", []any{id}},
		{"journal_lines", "TRUNCATE", "TRUNCATE journal_lines", nil},
		{"accounts", "UPDATE", "INSERT INTO accounts (name) VALUES ('cash') ON CONFLICT (name) DO UPDATE SET name = 'cash'", nil},
	}

	for _, tc := range tests {
		t.Run(tc.table+"/"+tc.op+"/"+tc.sql, func(t *testing.T) {
			_, err := conn.Exec(ctx, tc.sql, tc.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("error = %v, want *pgconn.PgError", err)
			}
			if pgErr.Code != "23001" {
				t.Errorf("SQLSTATE = %s, want 23001", pgErr.Code)
			}
			want := fmt.Sprintf("%s is append-only: %s is not allowed", tc.table, tc.op)
			if pgErr.Message != want {
				t.Errorf("message = %q, want %q", pgErr.Message, want)
			}
		})
	}

	// Nothing changed.
	var n int
	if err := conn.QueryRow(ctx,
		"SELECT count(*) FROM accounts WHERE name IN ('cash', 'revenue')").Scan(&n); err != nil || n != 2 {
		t.Errorf("accounts named cash/revenue = %d (err %v), want 2", n, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&n); err != nil || n != 2 {
		t.Errorf("accounts = %d (err %v), want 2", n, err)
	}
	if err := conn.QueryRow(ctx,
		"SELECT count(*) FROM journal_entries WHERE description = ''").Scan(&n); err != nil || n != 1 {
		t.Errorf("journal_entries with empty description = %d (err %v), want 1", n, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM journal_entries").Scan(&n); err != nil || n != 1 {
		t.Errorf("journal_entries = %d (err %v), want 1", n, err)
	}
	if err := conn.QueryRow(ctx,
		"SELECT count(*) FROM journal_lines WHERE entry_id = $1 AND amount_minor = 1000", id).Scan(&n); err != nil || n != 2 {
		t.Errorf("journal_lines for entry with amount 1000 = %d (err %v), want 2", n, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM journal_lines").Scan(&n); err != nil || n != 2 {
		t.Errorf("journal_lines = %d (err %v), want 2", n, err)
	}
}

func TestReversingEntry(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	id, err := postNamed(ctx, conn, "", originalLines)
	if err != nil {
		t.Fatalf("post original entry: %v", err)
	}
	_, err = postNamed(ctx, conn, fmt.Sprintf("reverses entry %d", id), []namedLine{
		{"cash", "credit", 1000, "USD"},
		{"revenue", "debit", 1000, "USD"},
	})
	if err != nil {
		t.Fatalf("reversing entry commit: %v", err)
	}

	var entries, lines, original int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM journal_entries").Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM journal_lines").Scan(&lines); err != nil {
		t.Fatal(err)
	}
	if entries != 2 || lines != 4 {
		t.Errorf("entries = %d, lines = %d, want 2 and 4", entries, lines)
	}
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM journal_lines l JOIN accounts a ON a.id = l.account_id
		 WHERE l.entry_id = $1 AND l.amount_minor = 1000 AND l.currency = 'USD'
		   AND ((a.name = 'cash' AND l.direction = 'debit') OR (a.name = 'revenue' AND l.direction = 'credit'))`,
		id).Scan(&original); err != nil {
		t.Fatal(err)
	}
	if original != 2 {
		t.Errorf("original entry lines unchanged = %d, want 2", original)
	}

	rows, err := conn.Query(ctx,
		`SELECT a.name, sum(CASE l.direction WHEN 'debit' THEN l.amount_minor ELSE -l.amount_minor END)
		 FROM journal_lines l JOIN accounts a ON a.id = l.account_id GROUP BY a.name`)
	if err != nil {
		t.Fatalf("balance query: %v", err)
	}
	defer rows.Close()
	got := map[string]int64{}
	for rows.Next() {
		var name string
		var net int64
		if err := rows.Scan(&name, &net); err != nil {
			t.Fatal(err)
		}
		got[name] = net
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cash", "revenue"} {
		net, ok := got[name]
		if !ok || net != 0 {
			t.Errorf("net balance of %s = %d (present %v), want 0", name, net, ok)
		}
	}
}
