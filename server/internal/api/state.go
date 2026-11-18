package api

import (
	"net/http"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// getState answers GET /api/state with {"version": n, "state": {...}}.
func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	snap, err := s.store.Snapshot(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// putState answers PUT /api/state. The body has the same shape as the GET response; the
// version is the one the client last saw, and the write only succeeds if it is still current.
func (s *Server) putState(w http.ResponseWriter, r *http.Request) {
	// Pointers tell "missing" from "zero": a missing version must not mean version 0.
	var body struct {
		Version *int64        `json:"version"`
		State   *kanban.State `json:"state"`
	}
	if err := decodeBody(w, r, maxStateBytes, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	if body.Version == nil {
		s.fail(w, r, badRequest("version is required"))
		return
	}
	if body.State == nil {
		s.fail(w, r, badRequest("state is required"))
		return
	}

	version, err := s.store.ReplaceState(r.Context(), *body.Version, *body.State)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"version": s.wrote(version)})
}
