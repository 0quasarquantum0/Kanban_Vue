// Package api holds the HTTP layer of the Kanban server: routes, handlers and middleware.
package api

import (
	"log/slog"
	"net/http"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// Server answers HTTP requests using a Store.
type Server struct {
	store kanban.Store
	log   *slog.Logger
}

// New returns a Server that keeps its data in store and logs to log.
func New(store kanban.Store, log *slog.Logger) *Server {
	return &Server{store: store, log: log}
}

// Handler returns the handler that serves every route of the API, wrapped in the middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/state", s.getState)
	mux.HandleFunc("PUT /api/state", s.putState)
	s.routeResources(mux)

	// The first wrapper is the outermost: logging sees the 500 that recoverPanics writes.
	return logRequests(s.log, recoverPanics(s.log, jsonRoutingErrors(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
