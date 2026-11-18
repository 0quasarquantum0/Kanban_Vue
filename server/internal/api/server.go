// Package api holds the HTTP layer of the Kanban server: routes, handlers and middleware.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// Server answers HTTP requests using a Store.
type Server struct {
	store     kanban.Store
	log       *slog.Logger
	hub       *Hub          // tells connected clients about new versions (see events.go)
	stopHub   func()        // ends the hub's goroutine
	heartbeat time.Duration // how often an idle event stream sends a keep-alive line
}

// New returns a Server that keeps its data in store and logs to log. It starts a goroutine for
// the event hub, so call Close when the server is no longer needed.
func New(store kanban.Store, log *slog.Logger) *Server {
	ctx, stop := context.WithCancel(context.Background())
	hub := NewHub(16)
	go hub.Run(ctx)
	return &Server{store: store, log: log, hub: hub, stopHub: stop, heartbeat: defaultHeartbeat}
}

// Close stops the event hub, which ends every open event stream.
func (s *Server) Close() { s.stopHub() }

// Handler returns the handler that serves every route of the API, wrapped in the middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/state", s.getState)
	mux.HandleFunc("PUT /api/state", s.putState)
	s.routeResources(mux)
	mux.HandleFunc("GET /api/events", s.events)

	// The first wrapper is the outermost: logging sees the 500 that recoverPanics writes.
	return logRequests(s.log, recoverPanics(s.log, jsonRoutingErrors(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
