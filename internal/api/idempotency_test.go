package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// writeCase describes how to exercise one write route.
type writeCase struct {
	// body builds a request that succeeds with wantStatus.
	body func(t *testing.T, conn *pgx.Conn) []byte
	// otherBody builds a different valid request for the same route.
	otherBody  func(t *testing.T, conn *pgx.Conn) []byte
	wantStatus int
}

// writeCases has one entry per write route, keyed by "METHOD PATH". Adding a
// write route without a case here fails TestWriteRoutesCovered.
var writeCases = map[string]writeCase{
	"POST /journal-entries": {
		body: func(t *testing.T, conn *pgx.Conn) []byte {
			return journalBody(t, conn, 1000)
		},
		otherBody: func(t *testing.T, conn *pgx.Conn) []byte {
			return journalBody(t, conn, 2000)
		},
		wantStatus: http.StatusCreated,
	},
}

// journalBody debits cash and credits revenue by amount USD.
func journalBody(t *testing.T, conn *pgx.Conn, amount int64) []byte {
	t.Helper()
	var cash, revenue int64
	err := conn.QueryRow(context.Background(),
		`SELECT (SELECT id FROM accounts WHERE name = 'cash'),
		        (SELECT id FROM accounts WHERE name = 'revenue')`).Scan(&cash, &revenue)
	if err != nil {
		t.Fatalf("look up accounts: %v", err)
	}
	return []byte(fmt.Sprintf(
		`{"description":"coffee","lines":[{"account_id":%d,"direction":"debit","amount_minor":%d,"currency":"USD"},{"account_id":%d,"direction":"credit","amount_minor":%d,"currency":"USD"}]}`,
		cash, amount, revenue, amount))
}

func isWriteMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// coverage returns write routes without a case and case keys with no
// matching write route. Both results are sorted.
func coverage(routes []Route, cases map[string]writeCase) (missing, stale []string) {
	writes := map[string]bool{}
	for _, r := range routes {
		if !isWriteMethod(r.Method) {
			continue
		}
		k := r.Method + " " + r.Path
		writes[k] = true
		if _, ok := cases[k]; !ok {
			missing = append(missing, k)
		}
	}
	for k := range cases {
		if !writes[k] {
			stale = append(stale, k)
		}
	}
	slices.Sort(missing)
	slices.Sort(stale)
	return missing, stale
}

func TestIdempotencyCoverageHelper(t *testing.T) {
	routes := []Route{{"POST", "/a"}, {"PUT", "/b"}, {"GET", "/c"}}
	cases := map[string]writeCase{"POST /a": {}, "DELETE /z": {}}
	missing, stale := coverage(routes, cases)
	if !slices.Equal(missing, []string{"PUT /b"}) {
		t.Errorf("missing = %v, want [PUT /b]", missing)
	}
	if !slices.Equal(stale, []string{"DELETE /z"}) {
		t.Errorf("stale = %v, want [DELETE /z]", stale)
	}
}

func TestWriteRoutesCovered(t *testing.T) {
	missing, stale := coverage(New(nil).Routes(), writeCases)
	for _, k := range missing {
		t.Errorf("write route %s has no idempotency test case", k)
	}
	for _, k := range stale {
		t.Errorf("idempotency test case %s matches no write route", k)
	}
}

func randomKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random key: %v", err)
	}
	return hex.EncodeToString(b)
}

// do sends one request through h. An empty key sends no Idempotency-Key header.
func do(h http.Handler, method, path, key string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const (
	keyRequiredJSON = `{"error":"idempotency_key_required","message":"Idempotency-Key header is required"}`
	keyReusedJSON   = `{"error":"idempotency_key_reused","message":"Idempotency-Key was already used with a different request"}`
)

func TestWriteRoutesIdempotent(t *testing.T) {
	conn := connect(t)
	router := New(conn)

	for _, route := range router.Routes() {
		if !isWriteMethod(route.Method) {
			continue
		}
		name := route.Method + " " + route.Path
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			tc, ok := writeCases[name]
			if !ok {
				t.Fatalf("write route %s has no idempotency test case", name)
			}
			body := tc.body(t, conn)
			key := randomKey(t)

			ledger0, idem0 := ledgerRows(t, conn), idempotencyRows(t, conn)
			assertCounts := func(step string, wantLedger, wantIdem int) {
				t.Helper()
				if got := ledgerRows(t, conn); got != wantLedger {
					t.Errorf("%s: ledger rows = %d, want %d", step, got, wantLedger)
				}
				if got := idempotencyRows(t, conn); got != wantIdem {
					t.Errorf("%s: idempotency rows = %d, want %d", step, got, wantIdem)
				}
			}

			// (c) missing key
			rec := do(router, route.Method, route.Path, "", body)
			if rec.Code != http.StatusBadRequest || rec.Body.String() != keyRequiredJSON {
				t.Errorf("no key: got %d %s", rec.Code, rec.Body.String())
			}
			assertCounts("no key", ledger0, idem0)

			// first request
			first := do(router, route.Method, route.Path, key, body)
			if first.Code != tc.wantStatus {
				t.Fatalf("first: status %d, want %d; body %s", first.Code, tc.wantStatus, first.Body.String())
			}
			if got := ledgerRows(t, conn); got <= ledger0 {
				t.Errorf("first: ledger rows = %d, want more than %d", got, ledger0)
			}
			if got := idempotencyRows(t, conn); got != idem0+1 {
				t.Errorf("first: idempotency rows = %d, want %d", got, idem0+1)
			}
			ledger1, idem1 := ledgerRows(t, conn), idempotencyRows(t, conn)

			checkReplay := func(step string) {
				t.Helper()
				rec := do(router, route.Method, route.Path, key, body)
				if rec.Code != first.Code || !bytes.Equal(rec.Body.Bytes(), first.Body.Bytes()) {
					t.Errorf("%s: got %d %q, want %d %q", step, rec.Code, rec.Body.Bytes(), first.Code, first.Body.Bytes())
				}
				assertCounts(step, ledger1, idem1)
			}

			// (a) replay
			checkReplay("replay")

			// (b) same key, different body
			rec = do(router, route.Method, route.Path, key, tc.otherBody(t, conn))
			if rec.Code != http.StatusUnprocessableEntity || rec.Body.String() != keyReusedJSON {
				t.Errorf("reused key: got %d %s", rec.Code, rec.Body.String())
			}
			assertCounts("reused key", ledger1, idem1)
			checkReplay("replay after reuse")
		})
	}
}
