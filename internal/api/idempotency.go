package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// anonymousClientID scopes every idempotency key until auth exists; auth will
// replace it with the authenticated client id.
const anonymousClientID = "anonymous"

const maxKeyBytes = 255

// writeHandler runs inside the idempotency transaction. It must not commit.
// A non-2xx status or an error rolls the transaction back.
type writeHandler func(ctx context.Context, tx pgx.Tx, body []byte) (status int, resp any, err error)

// apiError is an error with an HTTP status and a client-facing code and message.
type apiError struct {
	Status  int    `json:"-"`
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

var (
	errKeyRequired = &apiError{http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required"}
	errKeyInvalid  = &apiError{http.StatusBadRequest, "idempotency_key_invalid", "Idempotency-Key must be at most 255 bytes"}
	errKeyReused   = &apiError{http.StatusUnprocessableEntity, "idempotency_key_reused", "Idempotency-Key was already used with a different request"}
	errInternal    = &apiError{http.StatusInternalServerError, "internal", "internal error"}
)

// writeError maps err to a JSON error response. *apiError maps as given,
// SQLSTATE class 23 maps to 422, anything else is a 500 with no detail.
func writeError(w http.ResponseWriter, err error) {
	var ae *apiError
	var pe *pgconn.PgError
	switch {
	case errors.As(err, &ae):
	case errors.As(err, &pe) && len(pe.Code) >= 2 && pe.Code[:2] == "23":
		msg := "request violates a database constraint"
		if pe.Code == "23514" {
			msg = pe.Message
		}
		ae = &apiError{http.StatusUnprocessableEntity, "constraint_violation", msg}
	default:
		ae = errInternal
	}
	b, err := json.Marshal(ae)
	if err != nil {
		b, ae = []byte(`{"error":"internal","message":"internal error"}`), errInternal
	}
	writeRaw(w, ae.Status, b)
}

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}

// idempotent wraps h with the idempotency layer: the record is checked and
// inserted in the same transaction as the write, and the primary key on
// (client_id, idempotency_key) settles concurrent requests.
func (r *Router) idempotent(h writeHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := req.Header.Get("Idempotency-Key")
		if key == "" {
			writeError(w, errKeyRequired)
			return
		}
		if len(key) > maxKeyBytes {
			writeError(w, errKeyInvalid)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, err)
			return
		}
		hash := requestHash(req.Method, req.URL.Path, body)
		ctx := req.Context()

		tx, err := r.db.Begin(ctx)
		if err != nil {
			writeError(w, err)
			return
		}
		defer tx.Rollback(ctx)

		status, stored, found, err := lookup(ctx, tx, key, hash)
		if err != nil {
			writeError(w, err)
			return
		}
		if found {
			writeRaw(w, status, stored)
			return
		}

		status, resp, err := h(ctx, tx, body)
		if err != nil {
			writeError(w, err)
			return
		}
		respBody, err := json.Marshal(resp)
		if err != nil {
			writeError(w, err)
			return
		}
		if status < 200 || status > 299 {
			writeRaw(w, status, respBody)
			return
		}

		var one int
		err = tx.QueryRow(ctx,
			`INSERT INTO idempotency_keys (client_id, idempotency_key, request_hash, response_status, response_body)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (client_id, idempotency_key) DO NOTHING
			 RETURNING 1`,
			anonymousClientID, key, hash, status, respBody).Scan(&one)
		if errors.Is(err, pgx.ErrNoRows) {
			// A concurrent request with the same key committed first.
			tx.Rollback(ctx)
			r.replayCommitted(ctx, w, key, hash)
			return
		}
		if err != nil {
			writeError(w, err)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeError(w, err)
			return
		}
		writeRaw(w, status, respBody)
	})
}

// replayCommitted looks the key up in a fresh transaction and writes the
// stored response, or the 422 when the request differs.
func (r *Router) replayCommitted(ctx context.Context, w http.ResponseWriter, key string, hash []byte) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		writeError(w, err)
		return
	}
	defer tx.Rollback(ctx)
	status, stored, found, err := lookup(ctx, tx, key, hash)
	if err != nil {
		writeError(w, err)
		return
	}
	if !found {
		writeError(w, errInternal)
		return
	}
	writeRaw(w, status, stored)
}

// lookup finds the record for key. found is false when there is none. When
// the stored hash differs from hash it returns errKeyReused.
func lookup(ctx context.Context, tx pgx.Tx, key string, hash []byte) (status int, body []byte, found bool, err error) {
	var storedHash []byte
	err = tx.QueryRow(ctx,
		`SELECT request_hash, response_status, response_body
		   FROM idempotency_keys
		  WHERE client_id = $1 AND idempotency_key = $2`,
		anonymousClientID, key).Scan(&storedHash, &status, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	if !bytes.Equal(storedHash, hash) {
		return 0, nil, true, errKeyReused
	}
	return status, body, true, nil
}

func requestHash(method, path string, body []byte) []byte {
	h := sha256.New()
	h.Write([]byte(method + "\n" + path + "\n"))
	h.Write(body)
	return h.Sum(nil)
}
