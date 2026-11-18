package api

import (
	"context"
	"net/http"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// The replies to writes: the new version of the state, plus the thing that was created or
// changed when the client cannot know all of it (the server picks IDs and timestamps).
type (
	versionReply struct {
		Version int64 `json:"version"`
	}
	boardReply struct {
		Version int64        `json:"version"`
		Board   kanban.Board `json:"board"`
	}
	columnReply struct {
		Version int64         `json:"version"`
		Column  kanban.Column `json:"column"`
	}
	cardReply struct {
		Version int64       `json:"version"`
		Card    kanban.Card `json:"card"`
	}
)

// newItem is the body of every POST that creates something. The ID is optional: a client that
// updates its screen before the server answers can propose one, and the server keeps it when
// it is a valid, unused UUID.
type newItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// titleChange is the body of the PATCH that renames a board or a column.
type titleChange struct {
	Title string `json:"title"`
}

// cardPosition is the body of PUT /api/cards/{cardId}/position. Pointers make both required.
type cardPosition struct {
	ColumnID *string `json:"columnId"`
	Index    *int    `json:"index"`
}

func (s *Server) routeResources(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/boards", s.addBoard)
	mux.HandleFunc("PATCH /api/boards/{boardId}", s.rename("boardId", s.store.RenameBoard))
	mux.HandleFunc("DELETE /api/boards/{boardId}", s.remove("boardId", s.store.DeleteBoard))

	mux.HandleFunc("POST /api/boards/{boardId}/columns", s.addColumn)
	mux.HandleFunc("PATCH /api/columns/{columnId}", s.rename("columnId", s.store.RenameColumn))
	mux.HandleFunc("DELETE /api/columns/{columnId}", s.remove("columnId", s.store.DeleteColumn))

	mux.HandleFunc("POST /api/columns/{columnId}/cards", s.addCard)
	mux.HandleFunc("PATCH /api/cards/{cardId}", s.updateCard)
	mux.HandleFunc("DELETE /api/cards/{cardId}", s.remove("cardId", s.store.DeleteCard))
	mux.HandleFunc("PUT /api/cards/{cardId}/position", s.moveCard)
}

func (s *Server) addBoard(w http.ResponseWriter, r *http.Request) {
	var body newItem
	if err := decodeBody(w, r, maxBodyBytes, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	board, version, err := s.store.AddBoard(r.Context(), body.ID, body.Title)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, boardReply{Version: s.wrote(version), Board: board})
}

func (s *Server) addColumn(w http.ResponseWriter, r *http.Request) {
	var body newItem
	if err := decodeBody(w, r, maxBodyBytes, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	column, version, err := s.store.AddColumn(r.Context(), r.PathValue("boardId"), body.ID, body.Title)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, columnReply{Version: s.wrote(version), Column: column})
}

func (s *Server) addCard(w http.ResponseWriter, r *http.Request) {
	var body newItem
	if err := decodeBody(w, r, maxBodyBytes, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	card, version, err := s.store.AddCard(r.Context(), r.PathValue("columnId"), body.ID, body.Title)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, cardReply{Version: s.wrote(version), Card: card})
}

func (s *Server) updateCard(w http.ResponseWriter, r *http.Request) {
	var patch kanban.CardPatch
	if err := decodeBody(w, r, maxBodyBytes, &patch); err != nil {
		s.fail(w, r, err)
		return
	}
	card, version, err := s.store.UpdateCard(r.Context(), r.PathValue("cardId"), patch)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cardReply{Version: s.wrote(version), Card: card})
}

func (s *Server) moveCard(w http.ResponseWriter, r *http.Request) {
	var body cardPosition
	if err := decodeBody(w, r, maxBodyBytes, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	if body.ColumnID == nil || *body.ColumnID == "" {
		s.fail(w, r, badRequest("columnId is required"))
		return
	}
	if body.Index == nil {
		s.fail(w, r, badRequest("index is required"))
		return
	}
	version, err := s.store.MoveCard(r.Context(), r.PathValue("cardId"), *body.ColumnID, *body.Index)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, versionReply{Version: s.wrote(version)})
}

// rename builds the handler for a PATCH that changes a title. The same code serves boards and
// columns: they differ only in the path parameter and in which Store method does the work.
func (s *Server) rename(param string, change func(ctx context.Context, id, title string) (int64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body titleChange
		if err := decodeBody(w, r, maxBodyBytes, &body); err != nil {
			s.fail(w, r, err)
			return
		}
		version, err := change(r.Context(), r.PathValue(param), body.Title)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, versionReply{Version: s.wrote(version)})
	}
}

// remove builds the handler for a DELETE of one thing, named by the path parameter.
func (s *Server) remove(param string, del func(ctx context.Context, id string) (int64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, err := del(r.Context(), r.PathValue(param))
		if err != nil {
			s.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, versionReply{Version: s.wrote(version)})
	}
}
