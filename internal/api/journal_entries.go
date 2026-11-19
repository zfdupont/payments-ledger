package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/zfdupont/payments-ledger/internal/ledger"
)

type journalLineRequest struct {
	AccountID   int64  `json:"account_id"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type journalEntryRequest struct {
	Description string               `json:"description"`
	Lines       []journalLineRequest `json:"lines"`
}

var errInvalidBody = &apiError{http.StatusBadRequest, "invalid_body", "request body is not a valid journal entry"}

// postJournalEntry posts a journal entry through the ledger service. The
// balance check is the database's, at the idempotency layer's COMMIT.
func postJournalEntry(ctx context.Context, tx pgx.Tx, body []byte) (int, any, error) {
	var req journalEntryRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return 0, nil, errInvalidBody
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return 0, nil, errInvalidBody
	}

	e := ledger.Entry{Description: req.Description}
	for _, l := range req.Lines {
		e.Lines = append(e.Lines, ledger.Line(l))
	}
	id, err := ledger.PostEntry(ctx, tx, e)
	if err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}
