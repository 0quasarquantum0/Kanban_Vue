package api_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// reply is every field a successful write can send back; each endpoint fills some of them.
type reply struct {
	Version int64          `json:"version"`
	Board   *kanban.Board  `json:"board"`
	Column  *kanban.Column `json:"column"`
	Card    *kanban.Card   `json:"card"`
}

// send makes a request, checks the status, and decodes the JSON reply.
func send(t *testing.T, handler http.Handler, method, path, body string, wantStatus int) reply {
	t.Helper()
	rec := do(t, handler, method, path, body)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var r reply
	decode(t, rec, &r)
	return r
}

// world is a handler with one board holding two columns, each created through the API.
type world struct {
	t       *testing.T
	handler http.Handler
	board   kanban.Board
	todo    kanban.Column
	done    kanban.Column
}

func newWorld(t *testing.T) *world {
	t.Helper()
	handler := newHandler(t)
	w := &world{t: t, handler: handler}
	w.board = *send(t, handler, http.MethodPost, "/api/boards", `{"title":"Board"}`, http.StatusCreated).Board
	w.todo = *send(t, handler, http.MethodPost, "/api/boards/"+w.board.ID+"/columns", `{"title":"To do"}`, http.StatusCreated).Column
	w.done = *send(t, handler, http.MethodPost, "/api/boards/"+w.board.ID+"/columns", `{"title":"Done"}`, http.StatusCreated).Column
	return w
}

func (w *world) addCard(column kanban.Column, title string) kanban.Card {
	w.t.Helper()
	return *send(w.t, w.handler, http.MethodPost, "/api/columns/"+column.ID+"/cards", `{"title":"`+title+`"}`, http.StatusCreated).Card
}

func (w *world) state() kanban.State {
	w.t.Helper()
	return getSnapshot(w.t, w.handler).State
}

func (w *world) titles(columnID string) []string {
	w.t.Helper()
	state := w.state()
	titles := []string{}
	for _, id := range state.Columns[columnID].CardIDs {
		titles = append(titles, state.Cards[id].Title)
	}
	return titles
}

func TestPostBoardCreatesABoardAndReturnsIt(t *testing.T) {
	handler := newHandler(t)

	got := send(t, handler, http.MethodPost, "/api/boards", `{"title":"Roadmap"}`, http.StatusCreated)

	if got.Board == nil || got.Board.Title != "Roadmap" || !kanban.IsValidID(got.Board.ID) {
		t.Fatalf("board = %+v, want a titled board with a generated ID", got.Board)
	}
	if got.Board.ColumnIDs == nil {
		t.Error("columnIds is null; it must be an empty array")
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	snap := getSnapshot(t, handler)
	if snap.Version != 1 || snap.State.Boards[got.Board.ID].Title != "Roadmap" {
		t.Errorf("state after POST: version %d, boards %+v", snap.Version, snap.State.Boards)
	}
}

func TestPostAcceptsAClientChosenIDOnlyWhenItIsAFreeUUID(t *testing.T) {
	handler := newHandler(t)
	const wanted = "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6d"

	first := send(t, handler, http.MethodPost, "/api/boards", `{"id":"`+wanted+`","title":"A"}`, http.StatusCreated)
	second := send(t, handler, http.MethodPost, "/api/boards", `{"id":"`+wanted+`","title":"B"}`, http.StatusCreated)
	third := send(t, handler, http.MethodPost, "/api/boards", `{"id":"not-a-uuid","title":"C"}`, http.StatusCreated)

	if first.Board.ID != wanted {
		t.Errorf("first ID = %q, want the requested one", first.Board.ID)
	}
	for _, other := range []*kanban.Board{second.Board, third.Board} {
		if other.ID == wanted || other.ID == "not-a-uuid" || !kanban.IsValidID(other.ID) {
			t.Errorf("ID = %q, want a newly generated UUID", other.ID)
		}
	}
}

func TestCreatingThingsInsideOthers(t *testing.T) {
	w := newWorld(t)

	card := w.addCard(w.todo, "Write tests")

	state := w.state()
	if got := state.Boards[w.board.ID].ColumnIDs; !slices.Equal(got, []string{w.todo.ID, w.done.ID}) {
		t.Errorf("columnIds = %v, want both columns in creation order", got)
	}
	if got := state.Columns[w.todo.ID].CardIDs; !slices.Equal(got, []string{card.ID}) {
		t.Errorf("cardIds = %v, want the new card", got)
	}
	if card.Title != "Write tests" || card.DueDate != nil || card.Labels == nil || card.CreatedAt == "" {
		t.Errorf("card = %+v, want defaults filled in", card)
	}
}

func TestRenamingBoardsAndColumns(t *testing.T) {
	w := newWorld(t)

	send(t, w.handler, http.MethodPatch, "/api/boards/"+w.board.ID, `{"title":"Renamed board"}`, http.StatusOK)
	got := send(t, w.handler, http.MethodPatch, "/api/columns/"+w.todo.ID, `{"title":"Backlog"}`, http.StatusOK)

	state := w.state()
	if state.Boards[w.board.ID].Title != "Renamed board" || state.Columns[w.todo.ID].Title != "Backlog" {
		t.Errorf("titles after rename: %q, %q", state.Boards[w.board.ID].Title, state.Columns[w.todo.ID].Title)
	}
	if got.Version != getSnapshot(t, w.handler).Version {
		t.Errorf("the reply's version %d is not the current one", got.Version)
	}
}

func TestPatchCardChangesOnlyTheFieldsSent(t *testing.T) {
	w := newWorld(t)
	card := w.addCard(w.todo, "Original")

	got := send(t, w.handler, http.MethodPatch, "/api/cards/"+card.ID,
		`{"description":"More detail","labels":["bug","ui"],"dueDate":"2026-12-01"}`, http.StatusOK)

	if got.Card == nil || got.Card.Title != "Original" || got.Card.Description != "More detail" ||
		!slices.Equal(got.Card.Labels, []string{"bug", "ui"}) || got.Card.DueDate == nil || *got.Card.DueDate != "2026-12-01" {
		t.Fatalf("card in the reply = %+v", got.Card)
	}

	cleared := send(t, w.handler, http.MethodPatch, "/api/cards/"+card.ID, `{"dueDate":null}`, http.StatusOK)

	if cleared.Card.DueDate != nil {
		t.Errorf("dueDate = %v after sending null, want it cleared", *cleared.Card.DueDate)
	}
	if cleared.Card.Description != "More detail" {
		t.Errorf("description = %q, want it untouched by a patch that did not mention it", cleared.Card.Description)
	}
	if stored := w.state().Cards[card.ID]; stored.DueDate != nil || stored.Description != "More detail" {
		t.Errorf("stored card = %+v", stored)
	}
}

func TestMovingCards(t *testing.T) {
	w := newWorld(t)
	a, b, c := w.addCard(w.todo, "a"), w.addCard(w.todo, "b"), w.addCard(w.todo, "c")
	w.addCard(w.done, "x")

	t.Run("within a column", func(t *testing.T) {
		send(t, w.handler, http.MethodPut, "/api/cards/"+a.ID+"/position", `{"columnId":"`+w.todo.ID+`","index":2}`, http.StatusOK)
		if got := w.titles(w.todo.ID); !slices.Equal(got, []string{"b", "c", "a"}) {
			t.Errorf("todo = %v", got)
		}
	})
	t.Run("to another column at an index", func(t *testing.T) {
		send(t, w.handler, http.MethodPut, "/api/cards/"+b.ID+"/position", `{"columnId":"`+w.done.ID+`","index":0}`, http.StatusOK)
		if got := w.titles(w.done.ID); !slices.Equal(got, []string{"b", "x"}) {
			t.Errorf("done = %v", got)
		}
		if got := w.titles(w.todo.ID); !slices.Equal(got, []string{"c", "a"}) {
			t.Errorf("todo = %v", got)
		}
	})
	t.Run("a huge index appends", func(t *testing.T) {
		send(t, w.handler, http.MethodPut, "/api/cards/"+c.ID+"/position", `{"columnId":"`+w.done.ID+`","index":999}`, http.StatusOK)
		if got := w.titles(w.done.ID); !slices.Equal(got, []string{"b", "x", "c"}) {
			t.Errorf("done = %v", got)
		}
	})
}

func TestMoveRejectsBadRequests(t *testing.T) {
	w := newWorld(t)
	card := w.addCard(w.todo, "a")
	path := "/api/cards/" + card.ID + "/position"

	tests := []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"an unknown card", "/api/cards/nope/position", `{"columnId":"` + w.done.ID + `","index":0}`, http.StatusNotFound},
		{"an unknown column", path, `{"columnId":"nope","index":0}`, http.StatusNotFound},
		{"a missing column", path, `{"index":0}`, http.StatusBadRequest},
		{"a missing index", path, `{"columnId":"` + w.done.ID + `"}`, http.StatusBadRequest},
		{"an index that is not a whole number", path, `{"columnId":"` + w.done.ID + `","index":1.5}`, http.StatusBadRequest},
		{"an unknown field", path, `{"columnId":"` + w.done.ID + `","index":0,"x":1}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			wantError(t, do(t, w.handler, http.MethodPut, tt.path, tt.body), tt.status)
		})
	}
	if got := w.titles(w.todo.ID); !slices.Equal(got, []string{"a"}) {
		t.Errorf("a rejected move changed the card's place: %v", got)
	}
}

func TestDeletingCascades(t *testing.T) {
	w := newWorld(t)
	card := w.addCard(w.todo, "doomed")
	kept := w.addCard(w.done, "kept")

	send(t, w.handler, http.MethodDelete, "/api/cards/"+card.ID, "", http.StatusOK)
	if _, ok := w.state().Cards[card.ID]; ok {
		t.Error("the deleted card is still there")
	}

	doomed := w.addCard(w.todo, "also doomed")
	send(t, w.handler, http.MethodDelete, "/api/columns/"+w.todo.ID, "", http.StatusOK)
	state := w.state()
	if _, ok := state.Columns[w.todo.ID]; ok {
		t.Error("the deleted column is still there")
	}
	if _, ok := state.Cards[doomed.ID]; ok {
		t.Error("a card survived its column")
	}
	if got := state.Boards[w.board.ID].ColumnIDs; !slices.Equal(got, []string{w.done.ID}) {
		t.Errorf("columnIds = %v, want only the remaining column", got)
	}

	send(t, w.handler, http.MethodDelete, "/api/boards/"+w.board.ID, "", http.StatusOK)
	state = w.state()
	if len(state.Boards)+len(state.Columns)+len(state.Cards) != 0 {
		t.Errorf("state after deleting the board = %+v, want it empty (kept card %q included)", state, kept.ID)
	}
}

func TestUnknownIDsAre404(t *testing.T) {
	w := newWorld(t)

	tests := []struct{ method, path, body string }{
		{http.MethodPatch, "/api/boards/nope", `{"title":"x"}`},
		{http.MethodDelete, "/api/boards/nope", ""},
		{http.MethodPost, "/api/boards/nope/columns", `{"title":"x"}`},
		{http.MethodPatch, "/api/columns/nope", `{"title":"x"}`},
		{http.MethodDelete, "/api/columns/nope", ""},
		{http.MethodPost, "/api/columns/nope/cards", `{"title":"x"}`},
		{http.MethodPatch, "/api/cards/nope", `{"title":"x"}`},
		{http.MethodDelete, "/api/cards/nope", ""},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			message := wantError(t, do(t, w.handler, tt.method, tt.path, tt.body), http.StatusNotFound)
			if !strings.Contains(message, "nope") {
				t.Errorf("message %q does not name what was missing", message)
			}
		})
	}
}

func TestResourceEndpointsRejectBadBodies(t *testing.T) {
	w := newWorld(t)
	card := w.addCard(w.todo, "a")

	tests := []struct {
		name         string
		method, path string
		body         string
	}{
		{"an empty board title", http.MethodPost, "/api/boards", `{"title":"  "}`},
		{"a missing board title", http.MethodPost, "/api/boards", `{}`},
		{"an unknown field on a new board", http.MethodPost, "/api/boards", `{"title":"x","colour":"red"}`},
		{"a title that is not a string", http.MethodPost, "/api/boards", `{"title":5}`},
		{"no body at all", http.MethodPost, "/api/boards/" + w.board.ID + "/columns", ``},
		{"an empty column title", http.MethodPatch, "/api/columns/" + w.todo.ID, `{"title":""}`},
		{"an empty card title on create", http.MethodPost, "/api/columns/" + w.todo.ID + "/cards", `{"title":""}`},
		{"an empty card title on patch", http.MethodPatch, "/api/cards/" + card.ID, `{"title":""}`},
		{"labels that are not a list", http.MethodPatch, "/api/cards/" + card.ID, `{"labels":"bug"}`},
		{"an unknown card field", http.MethodPatch, "/api/cards/" + card.ID, `{"createdAt":"x"}`},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			wantError(t, do(t, w.handler, tt.method, tt.path, tt.body), http.StatusBadRequest)
		})
	}
}

func TestEveryWriteRaisesTheVersionByOne(t *testing.T) {
	w := newWorld(t) // three writes so far
	card := w.addCard(w.todo, "a")

	steps := []struct{ method, path, body string }{
		{http.MethodPatch, "/api/cards/" + card.ID, `{"title":"b"}`},
		{http.MethodPut, "/api/cards/" + card.ID + "/position", `{"columnId":"` + w.done.ID + `","index":0}`},
		{http.MethodDelete, "/api/cards/" + card.ID, ""},
	}
	want := getSnapshot(t, w.handler).Version
	for _, step := range steps {
		want++
		got := send(t, w.handler, step.method, step.path, step.body, http.StatusOK)
		if got.Version != want {
			t.Errorf("%s %s: version %d, want %d", step.method, step.path, got.Version, want)
		}
	}
}
