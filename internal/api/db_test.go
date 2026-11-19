package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// connect returns a connection whose search_path is a throwaway schema with
// every up migration applied in sorted order and accounts cash and revenue
// seeded. It skips when Postgres is not available.
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

// ledgerRows is the total row count across the ledger tables.
func ledgerRows(t *testing.T, conn *pgx.Conn) int {
	t.Helper()
	var n int
	err := conn.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM accounts)
		      + (SELECT count(*) FROM journal_entries)
		      + (SELECT count(*) FROM journal_lines)`).Scan(&n)
	if err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	return n
}

// idempotencyRows is the row count of idempotency_keys.
func idempotencyRows(t *testing.T, conn *pgx.Conn) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), "SELECT count(*) FROM idempotency_keys").Scan(&n); err != nil {
		t.Fatalf("count idempotency rows: %v", err)
	}
	return n
}
