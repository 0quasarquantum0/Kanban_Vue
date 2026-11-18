// Package api holds the HTTP layer of the Kanban server: routes, handlers and middleware.
package api

import (
	"encoding/json"
	"net/http"
)

// NewHandler returns the handler that serves every route of the API.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", health)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Encode can only fail if the client went away, and there is nothing useful to do then.
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
