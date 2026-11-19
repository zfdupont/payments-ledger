// Package api is the HTTP layer. Every write route is registered through
// Router.addWrite, which always applies the idempotency layer.
package api

import (
	"context"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"
)

// TxBeginner starts a transaction. Both *pgx.Conn and *pgxpool.Pool satisfy it.
type TxBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// Route is a registered HTTP method and path.
type Route struct {
	Method, Path string
}

// Router is the HTTP router. It lists its routes so tests can check that
// every write route is covered.
type Router struct {
	db     TxBeginner
	mux    *http.ServeMux
	routes []Route
}

// New returns a Router backed by db. db may be nil when only Routes is used.
func New(db TxBeginner) *Router {
	r := &Router{db: db, mux: http.NewServeMux()}
	r.addWrite(http.MethodPost, "/journal-entries", postJournalEntry)
	return r
}

// Routes returns a copy of the registered routes.
func (r *Router) Routes() []Route {
	return slices.Clone(r.routes)
}

// ServeHTTP delegates to the mux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// addWrite is the only way to register a write route. It always wraps h with
// the idempotency layer.
func (r *Router) addWrite(method, path string, h writeHandler) {
	r.routes = append(r.routes, Route{Method: method, Path: path})
	r.mux.Handle(method+" "+path, r.idempotent(h))
}
