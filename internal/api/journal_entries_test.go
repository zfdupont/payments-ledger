package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

type errBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func decodeErr(t *testing.T, raw []byte) errBody {
	t.Helper()
	var e errBody
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("decode error body %q: %v", raw, err)
	}
	return e
}

func TestPostJournalEntryCreatesRows(t *testing.T) {
	conn := connect(t)
	router := New(conn)
	body := journalBody(t, conn, 1000)

	rec := do(router, http.MethodPost, "/journal-entries", randomKey(t), body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.ID == 0 {
		t.Fatalf("response %q: id=%d err=%v", rec.Body.String(), resp.ID, err)
	}

	ctx := context.Background()
	var entries int
	var desc string
	err := conn.QueryRow(ctx, "SELECT count(*), coalesce(max(description), '') FROM journal_entries WHERE id = $1", resp.ID).Scan(&entries, &desc)
	if err != nil || entries != 1 || desc != "coffee" {
		t.Fatalf("entry rows = %d desc = %q err = %v", entries, desc, err)
	}
	if got := ledgerRows(t, conn) - 2; got != 3 { // minus seeded accounts: 1 entry + 2 lines
		t.Errorf("ledger rows beyond accounts = %d, want 3", got)
	}

	rows, err := conn.Query(ctx,
		`SELECT (SELECT name FROM accounts WHERE id = account_id), direction, amount_minor, currency
		   FROM journal_lines WHERE entry_id = $1 ORDER BY id`, resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var name, dir, cur string
		var amt int64
		if err := rows.Scan(&name, &dir, &amt, &cur); err != nil {
			t.Fatal(err)
		}
		got = append(got, fmt.Sprintf("%s %s %d %s", name, dir, amt, cur))
	}
	want := []string{"cash debit 1000 USD", "revenue credit 1000 USD"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestPostJournalEntryUnbalanced(t *testing.T) {
	conn := connect(t)
	router := New(conn)
	good := journalBody(t, conn, 1000)
	bad := []byte(strings.Replace(string(good), `"amount_minor":1000,"currency":"USD"}]`, `"amount_minor":999,"currency":"USD"}]`, 1))
	if string(bad) == string(good) {
		t.Fatal("failed to build unbalanced body")
	}
	key := randomKey(t)
	l0, i0 := ledgerRows(t, conn), idempotencyRows(t, conn)

	rec := do(router, http.MethodPost, "/journal-entries", key, bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	e := decodeErr(t, rec.Body.Bytes())
	if e.Error != "constraint_violation" || !regexp.MustCompile(`^journal entry \d+ is unbalanced in USD: debits 1000 <> credits 999$`).MatchString(e.Message) {
		t.Errorf("error body = %+v", e)
	}
	if ledgerRows(t, conn) != l0 || idempotencyRows(t, conn) != i0 {
		t.Errorf("rows changed after rejected entry: ledger %d->%d, idempotency %d->%d",
			l0, ledgerRows(t, conn), i0, idempotencyRows(t, conn))
	}

	rec = do(router, http.MethodPost, "/journal-entries", key, good)
	if rec.Code != http.StatusCreated {
		t.Errorf("retry with balanced body: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestPostJournalEntryUnknownAccount(t *testing.T) {
	conn := connect(t)
	router := New(conn)
	body := []byte(`{"lines":[{"account_id":999999,"direction":"debit","amount_minor":5,"currency":"USD"},{"account_id":999999,"direction":"credit","amount_minor":5,"currency":"USD"}]}`)
	l0, i0 := ledgerRows(t, conn), idempotencyRows(t, conn)

	rec := do(router, http.MethodPost, "/journal-entries", randomKey(t), body)
	e := decodeErr(t, rec.Body.Bytes())
	if rec.Code != http.StatusUnprocessableEntity || e.Error != "constraint_violation" || e.Message != "request violates a database constraint" {
		t.Errorf("got %d %+v", rec.Code, e)
	}
	if ledgerRows(t, conn) != l0 || idempotencyRows(t, conn) != i0 {
		t.Error("rows changed after rejected entry")
	}
}

func TestPostJournalEntryInvalidBody(t *testing.T) {
	conn := connect(t)
	router := New(conn)
	good := string(journalBody(t, conn, 1000))
	cases := map[string]string{
		"fractional amount": strings.Replace(good, `"amount_minor":1000`, `"amount_minor":10.5`, 1),
		"unknown field":     strings.Replace(good, `{"description"`, `{"extra":1,"description"`, 1),
		"bad json":          `{"lines":`,
		"trailing data":     good + ` {}`,
	}
	for name, body := range cases {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			if name != "bad json" && body == good {
				t.Fatal("failed to build body")
			}
			l0, i0 := ledgerRows(t, conn), idempotencyRows(t, conn)
			rec := do(router, http.MethodPost, "/journal-entries", randomKey(t), []byte(body))
			want := `{"error":"invalid_body","message":"request body is not a valid journal entry"}`
			if rec.Code != http.StatusBadRequest || rec.Body.String() != want {
				t.Errorf("got %d %s", rec.Code, rec.Body.String())
			}
			if ledgerRows(t, conn) != l0 || idempotencyRows(t, conn) != i0 {
				t.Error("rows changed after invalid body")
			}
		})
	}
}

func TestPostJournalEntryKeyTooLong(t *testing.T) {
	conn := connect(t)
	router := New(conn)
	l0, i0 := ledgerRows(t, conn), idempotencyRows(t, conn)

	rec := do(router, http.MethodPost, "/journal-entries", strings.Repeat("k", 256), journalBody(t, conn, 1000))
	want := `{"error":"idempotency_key_invalid","message":"Idempotency-Key must be at most 255 bytes"}`
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want {
		t.Errorf("got %d %s", rec.Code, rec.Body.String())
	}
	if ledgerRows(t, conn) != l0 || idempotencyRows(t, conn) != i0 {
		t.Error("rows changed after invalid key")
	}
}
