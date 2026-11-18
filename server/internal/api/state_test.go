package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// seededHandler is a handler whose store already holds the demo board (version 1).
func seededHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, store := newHandlerWithStore(t)
	if err := kanban.EnsureSeeded(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	return handler
}

func getSnapshot(t *testing.T, handler http.Handler) kanban.Snapshot {
	t.Helper()
	rec := do(t, handler, http.MethodGet, "/api/state", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state = %d: %s", rec.Code, rec.Body.String())
	}
	var snap kanban.Snapshot
	decode(t, rec, &snap)
	return snap
}

// putBody builds the request body for PUT /api/state.
func putBody(t *testing.T, version int64, state kanban.State) string {
	t.Helper()
	data, err := json.Marshal(kanban.Snapshot{Version: version, State: state})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGetStateReturnsTheVersionAndTheState(t *testing.T) {
	handler := seededHandler(t)

	rec := do(t, handler, http.MethodGet, "/api/state", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var snap kanban.Snapshot
	decode(t, rec, &snap)
	if snap.Version != 1 {
		t.Errorf("version = %d, want 1", snap.Version)
	}
	if err := kanban.Validate(snap.State); err != nil {
		t.Errorf("the state is not valid: %v", err)
	}
	if len(snap.State.Boards) != 1 || len(snap.State.Cards) != 6 {
		t.Errorf("got %d boards and %d cards, want the seed's 1 and 6", len(snap.State.Boards), len(snap.State.Cards))
	}
}

func TestGetStateUsesTheFrontendJSONShape(t *testing.T) {
	rec := do(t, seededHandler(t), http.MethodGet, "/api/state", "")

	var raw struct {
		Version int                        `json:"version"`
		State   map[string]json.RawMessage `json:"state"`
	}
	decode(t, rec, &raw)

	for _, key := range []string{"boards", "columns", "cards"} {
		if _, ok := raw.State[key]; !ok {
			t.Errorf("state has no %q key", key)
		}
	}
	if !strings.Contains(rec.Body.String(), `"cardIds":[`) || !strings.Contains(rec.Body.String(), `"dueDate":null`) {
		t.Errorf("body does not look like the frontend's KanbanState: %s", rec.Body.String())
	}
}

func TestPutStateReplacesTheStateAndReturnsTheNewVersion(t *testing.T) {
	handler := seededHandler(t)
	next := kanban.NewSeedState()
	next.Boards[next.BoardOrder[0]] = kanban.Board{ID: next.BoardOrder[0], Title: "Replaced", ColumnIDs: next.Boards[next.BoardOrder[0]].ColumnIDs}

	rec := do(t, handler, http.MethodPut, "/api/state", putBody(t, 1, next))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var reply struct {
		Version int64 `json:"version"`
	}
	decode(t, rec, &reply)
	if reply.Version != 2 {
		t.Errorf("version = %d, want 2", reply.Version)
	}
	snap := getSnapshot(t, handler)
	if snap.Version != 2 || snap.State.Boards[next.BoardOrder[0]].Title != "Replaced" {
		t.Errorf("GET after PUT: version %d, board %+v", snap.Version, snap.State.Boards[next.BoardOrder[0]])
	}
}

func TestPutStateKeepsTheOrderOfBoards(t *testing.T) {
	handler := seededHandler(t)
	next := kanban.NewSeedState()
	ids := []string{kanban.NewID(), next.BoardOrder[0], kanban.NewID()}
	for _, id := range []string{ids[0], ids[2]} {
		next.Boards[id] = kanban.Board{ID: id, Title: id, ColumnIDs: []string{}}
	}
	next.BoardOrder = ids

	if rec := do(t, handler, http.MethodPut, "/api/state", putBody(t, 1, next)); rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rec.Code, rec.Body.String())
	}

	if got := getSnapshot(t, handler).State.BoardIDs(); !slices.Equal(got, ids) {
		t.Errorf("board order = %v, want %v", got, ids)
	}
}

func TestPutStateWithAStaleVersionIsAConflict(t *testing.T) {
	handler := seededHandler(t)
	before := getSnapshot(t, handler)

	rec := do(t, handler, http.MethodPut, "/api/state", putBody(t, 0, kanban.NewSeedState()))

	message := wantError(t, rec, http.StatusConflict)
	if !strings.Contains(message, "version") {
		t.Errorf("message %q does not mention the version", message)
	}
	after := getSnapshot(t, handler)
	if after.Version != before.Version || len(after.State.Cards) != len(before.State.Cards) {
		t.Errorf("a rejected PUT changed the store: %+v", after.Version)
	}
}

func TestPutStateRejectsBadRequests(t *testing.T) {
	handler := seededHandler(t)
	validState := string(mustMarshal(t, kanban.NewSeedState()))

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string // a fragment of the error message
	}{
		{"an empty body", ``, http.StatusBadRequest, "empty"},
		{"malformed JSON", `{"version":`, http.StatusBadRequest, "JSON"},
		{"text after the JSON", `{"version":1,"state":` + validState + `} x`, http.StatusBadRequest, "after"},
		{"a missing version", `{"state":` + validState + `}`, http.StatusBadRequest, "version"},
		{"a missing state", `{"version":1}`, http.StatusBadRequest, "state"},
		{"a null state", `{"version":1,"state":null}`, http.StatusBadRequest, "state"},
		{"an unknown field", `{"version":1,"state":` + validState + `,"extra":true}`, http.StatusBadRequest, "extra"},
		{"a version that is not a number", `{"version":"1","state":` + validState + `}`, http.StatusBadRequest, "version"},
		{"a state of the wrong shape", `{"version":1,"state":{"boards":[]}}`, http.StatusBadRequest, ""},
		{"a state that points at a missing card",
			`{"version":1,"state":{"boards":{"b":{"id":"b","title":"t","columnIds":["c"]}},"columns":{"c":{"id":"c","title":"t","cardIds":["x"]}},"cards":{}}}`,
			http.StatusBadRequest, "x"},
		{"a body over the size limit", `{"version":1,"state":` + validState + `}` + strings.Repeat(" ", 17<<20),
			http.StatusRequestEntityTooLarge, "large"},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			rec := do(t, handler, http.MethodPut, "/api/state", tt.body)

			message := wantError(t, rec, tt.wantStatus)
			if !strings.Contains(message, tt.wantMessage) {
				t.Errorf("message %q does not contain %q", message, tt.wantMessage)
			}
		})
	}

	if got := getSnapshot(t, handler).Version; got != 1 {
		t.Errorf("version = %d after only bad requests, want 1", got)
	}
}

func TestStoreFailuresBecomeJSON500s(t *testing.T) {
	failing := failingStore{Store: kanban.NewMemory()}
	rec := do(t, newHandlerOn(t, failing), http.MethodGet, "/api/state", "")

	message := wantError(t, rec, http.StatusInternalServerError)
	if strings.Contains(message, "disk on fire") {
		t.Errorf("message %q leaks an internal error to the client", message)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
