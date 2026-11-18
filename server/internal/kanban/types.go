// Package kanban holds the data model of the Kanban board and the Store that keeps it.
//
// The types mirror src/types.ts in the Vue app, so the JSON they produce is exactly what the
// frontend's KanbanState (and its Export JSON format) looks like.
package kanban

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
)

// Card is one task on the board.
type Card struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	DueDate     *string  `json:"dueDate"` // an ISO date, or nil (JSON null) for no due date
	CreatedAt   string   `json:"createdAt"`
}

// Column is an ordered list of cards.
type Column struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	CardIDs []string `json:"cardIds"` // order matters
}

// Board is an ordered list of columns.
type Board struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	ColumnIDs []string `json:"columnIds"` // order matters
}

// State is everything the app knows: entities stored by ID, order stored as ID slices.
type State struct {
	Boards  map[string]Board  `json:"boards"`
	Columns map[string]Column `json:"columns"`
	Cards   map[string]Card   `json:"cards"`

	// BoardOrder lists the board IDs in display order. The frontend shows boards in the
	// order of the keys in the JSON object, but a Go map has no order, so the order travels
	// separately. It is never a JSON field of its own: MarshalJSON writes the boards in this
	// order and UnmarshalJSON reads it back from the key order.
	BoardOrder []string `json:"-"`
}

// BoardIDs returns the board IDs in display order: those listed in BoardOrder that exist,
// then any remaining ones sorted by ID. It always covers every board exactly once.
func (s State) BoardIDs() []string {
	ids := make([]string, 0, len(s.Boards))
	seen := make(map[string]bool, len(s.Boards))
	for _, id := range s.BoardOrder {
		if _, ok := s.Boards[id]; ok && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	for _, id := range slices.Sorted(maps.Keys(s.Boards)) {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// MarshalJSON writes the state with the boards in display order and with nil maps as {}.
func (s State) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"boards":{`)
	for i, id := range s.BoardIDs() {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(id)
		if err != nil {
			return nil, err
		}
		board, err := json.Marshal(s.Boards[id])
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(board)
	}
	buf.WriteString(`},"columns":`)
	if err := writeMap(&buf, s.Columns); err != nil {
		return nil, err
	}
	buf.WriteString(`,"cards":`)
	if err := writeMap(&buf, s.Cards); err != nil {
		return nil, err
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// writeMap encodes m, writing a nil map as {} instead of null.
func writeMap[V any](buf *bytes.Buffer, m map[string]V) error {
	if m == nil {
		buf.WriteString("{}")
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	buf.Write(data)
	return nil
}

// UnmarshalJSON reads a state, remembering the order of the board keys. Unknown fields are
// errors. A missing or null "boards", "columns" or "cards" is left nil; Validate reports it.
func (s *State) UnmarshalJSON(data []byte) error {
	var raw struct {
		Boards  json.RawMessage   `json:"boards"`
		Columns map[string]Column `json:"columns"`
		Cards   map[string]Card   `json:"cards"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	boards, order, err := decodeBoards(raw.Boards)
	if err != nil {
		return err
	}
	*s = State{Boards: boards, Columns: raw.Columns, Cards: raw.Cards, BoardOrder: order}
	return nil
}

// decodeBoards reads a JSON object of boards token by token, which is the only way to see the
// order of its keys.
func decodeBoards(raw json.RawMessage) (map[string]Board, []string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()

	if tok, err := dec.Token(); err != nil {
		return nil, nil, err
	} else if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, nil, fmt.Errorf("boards must be an object, got %v", tok)
	}

	boards := map[string]Board{}
	order := []string{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key, _ := keyToken.(string) // object keys are always strings
		var board Board
		if err := dec.Decode(&board); err != nil {
			return nil, nil, err
		}
		if _, dup := boards[key]; !dup {
			order = append(order, key)
		}
		boards[key] = board
	}
	if _, err := dec.Token(); err != nil && err != io.EOF { // the closing brace
		return nil, nil, err
	}
	return boards, order, nil
}
