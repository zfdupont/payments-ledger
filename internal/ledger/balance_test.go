package ledger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// connect returns a connection whose search_path is a throwaway schema with
// every up migration applied in sorted order. It skips when Postgres is not available.
func connect(t *testing.T) *pgx.Conn {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, url)
	if err == nil {
		err = conn.Ping(ctx)
		if err != nil {
			conn.Close(context.Background())
		}
	}
	if err != nil {
		t.Skipf("Postgres unreachable at DATABASE_URL: %v", err)
	}

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		conn.Close(context.Background())
		t.Fatalf("random schema suffix: %v", err)
	}
	schema := "ledger_test_" + hex.EncodeToString(suffix)

	t.Cleanup(func() {
		bg := context.Background()
		if _, err := conn.Exec(bg, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
		}
		conn.Close(bg)
	})

	bg := context.Background()
	if _, err := conn.Exec(bg, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := conn.Exec(bg, "SET search_path TO "+schema); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	files, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("find migrations: %v (matched %d files)", err, len(files))
	}
	slices.Sort(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := conn.Exec(bg, string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", f, err)
		}
	}
	if _, err := conn.Exec(bg, "INSERT INTO accounts (name) VALUES ('cash'), ('revenue')"); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}
	return conn
}

type line struct {
	direction string
	amount    int64
	currency  string
}

// post inserts an entry and its lines in one transaction. It returns the
// entry id, the first insert error, and the commit error separately.
func post(ctx context.Context, conn *pgx.Conn, lines []line) (id int64, insertErr, commitErr error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err, nil
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, "INSERT INTO journal_entries DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
		return 0, err, nil
	}
	for _, l := range lines {
		_, err := tx.Exec(ctx,
			`INSERT INTO journal_lines (entry_id, account_id, direction, amount_minor, currency)
			 VALUES ($1, (SELECT id FROM accounts WHERE name = $2), $3, $4, $5)`,
			id, accountFor(l.direction), l.direction, l.amount, l.currency)
		if err != nil {
			return id, err, nil
		}
	}
	return id, nil, tx.Commit(ctx)
}

func accountFor(direction string) string {
	if direction == "debit" {
		return "cash"
	}
	return "revenue"
}

func TestBalance(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		lines   []line
		wantMsg string // empty means the commit must succeed
	}{
		{
			name:  "balanced",
			lines: []line{{"debit", 1000, "USD"}, {"credit", 1000, "USD"}},
		},
		{
			name: "balanced_multi_currency",
			lines: []line{
				{"debit", 1000, "USD"}, {"credit", 1000, "USD"},
				{"debit", 50, "EUR"}, {"credit", 50, "EUR"},
			},
		},
		{
			name:    "unbalanced",
			lines:   []line{{"debit", 1000, "USD"}, {"credit", 999, "USD"}},
			wantMsg: "is unbalanced in USD: debits 1000 <> credits 999",
		},
		{
			name:    "one_sided",
			lines:   []line{{"debit", 1000, "USD"}},
			wantMsg: "is unbalanced in USD: debits 1000 <> credits 0",
		},
		{
			name:    "balanced_total_not_per_currency",
			lines:   []line{{"debit", 1000, "USD"}, {"credit", 1000, "EUR"}},
			wantMsg: "is unbalanced in EUR: debits 0 <> credits 1000",
		},
		{
			name:    "no_lines",
			wantMsg: "has no lines",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, insertErr, commitErr := post(ctx, conn, tc.lines)
			if insertErr != nil {
				t.Fatalf("insert returned error (check must be deferred): %v", insertErr)
			}

			var count int
			if err := conn.QueryRow(ctx, "SELECT count(*) FROM journal_entries WHERE id = $1", id).Scan(&count); err != nil {
				t.Fatalf("count entry: %v", err)
			}

			if tc.wantMsg == "" {
				if commitErr != nil {
					t.Fatalf("commit failed: %v", commitErr)
				}
				if count != 1 {
					t.Fatalf("entry rows = %d, want 1", count)
				}
				return
			}

			var pgErr *pgconn.PgError
			if !errors.As(commitErr, &pgErr) {
				t.Fatalf("commit error = %v, want *pgconn.PgError", commitErr)
			}
			if pgErr.Code != "23514" {
				t.Errorf("SQLSTATE = %s, want 23514", pgErr.Code)
			}
			want := fmt.Sprintf("journal entry %d %s", id, tc.wantMsg)
			if pgErr.Message != want {
				t.Errorf("message = %q, want %q", pgErr.Message, want)
			}
			t.Log(pgErr.Message)
			if count != 0 {
				t.Errorf("entry rows = %d after failed commit, want 0", count)
			}
		})
	}
}
