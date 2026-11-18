package kanban

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// seedCopy is the seed after a JSON round trip, as it would arrive from an imported file.
func seedCopy(t *testing.T) State {
	t.Helper()
	data, err := json.Marshal(NewSeedState())
	if err != nil {
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

// parse decodes JSON the way the HTTP layer will and then validates it, so a wrongly typed
// field and a wrongly shaped state are both "errors" here, as isKanbanState() in
// src/validation.ts treats them both as "false".
func parse(data string) error {
	var state State
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return err
	}
	return Validate(state)
}

func TestValidateAcceptsTheSeedAndAnEmptyState(t *testing.T) {
	if err := Validate(seedCopy(t)); err != nil {
		t.Errorf("seed: %v", err)
	}
	if err := parse(`{"boards":{},"columns":{},"cards":{}}`); err != nil {
		t.Errorf("empty state: %v", err)
	}
}

func TestValidateRejectsWhatIsNotAState(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"null", `null`},
		{"a string", `"text"`},
		{"a number", `42`},
		{"an array", `[]`},
		{"an empty object", `{}`},
		{"a state without cards", `{"boards":{},"columns":{}}`},
		{"boards as an array", `{"boards":[],"columns":{},"cards":{}}`},
		{"an unknown top-level field", `{"boards":{},"columns":{},"cards":{},"extra":1}`},
		{"a card whose labels are a string", `{"boards":{},"columns":{},"cards":{"c":` +
			`{"id":"c","title":"t","description":"","labels":"bug","dueDate":null,"createdAt":""}}}`},
		{"a card whose labels are null", `{"boards":{},"columns":{},"cards":{"c":` +
			`{"id":"c","title":"t","description":"","labels":null,"dueDate":null,"createdAt":""}}}`},
		{"a column without cardIds", `{"boards":{},"columns":{"k":{"id":"k","title":"t"}},"cards":{}}`},
		{"a board without columnIds", `{"boards":{"b":{"id":"b","title":"t"}},"columns":{},"cards":{}}`},
		{"a card with an unknown field", `{"boards":{},"columns":{},"cards":{"c":` +
			`{"id":"c","title":"t","description":"","labels":[],"dueDate":null,"createdAt":"","x":1}}}`},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			if err := parse(tt.json); err == nil {
				t.Errorf("parse(%s) succeeded, want an error", tt.json)
			}
		})
	}
}

func TestValidateRejectsBrokenReferences(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(s *State)
		message string // a fragment the error must contain
	}{
		{
			name: "an entity stored under a different key",
			mutate: func(s *State) {
				for id, column := range s.Columns {
					delete(s.Columns, id)
					s.Columns["other-key"] = column
					return
				}
			},
			message: "other-key",
		},
		{
			name: "a column that points at a missing card",
			mutate: func(s *State) {
				for id, column := range s.Columns {
					column.CardIDs = append(column.CardIDs, "missing")
					s.Columns[id] = column
					return
				}
			},
			message: "missing",
		},
		{
			name: "a card placed in two columns",
			mutate: func(s *State) {
				var first Column
				for _, column := range s.Columns {
					if len(column.CardIDs) > 0 {
						first = column
						break
					}
				}
				for id, column := range s.Columns {
					if column.ID != first.ID {
						column.CardIDs = append(column.CardIDs, first.CardIDs[0])
						s.Columns[id] = column
						return
					}
				}
			},
			message: "more than once",
		},
		{
			name: "a board that points at a missing column",
			mutate: func(s *State) {
				for id, board := range s.Boards {
					board.ColumnIDs = append(board.ColumnIDs, "missing")
					s.Boards[id] = board
				}
			},
			message: "missing",
		},
		{
			name: "a column that no board lists",
			mutate: func(s *State) {
				for id, board := range s.Boards {
					board.ColumnIDs = board.ColumnIDs[1:]
					s.Boards[id] = board
				}
			},
			message: "not listed by any board",
		},
		{
			name: "a card that no column lists",
			mutate: func(s *State) {
				for id, column := range s.Columns {
					if len(column.CardIDs) > 0 {
						column.CardIDs = column.CardIDs[1:]
						s.Columns[id] = column
						return
					}
				}
			},
			message: "not listed by any column",
		},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			state := seedCopy(t)
			tt.mutate(&state)

			err := Validate(state)

			if err == nil {
				t.Fatal("Validate succeeded, want an error")
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("error %q does not match ErrInvalid", err)
			}
			if !strings.Contains(err.Error(), tt.message) {
				t.Errorf("error %q does not mention %q", err, tt.message)
			}
		})
	}
}

func TestValidateRejectsAMissingSection(t *testing.T) {
	state := seedCopy(t)
	state.Cards = nil

	err := Validate(state)

	if err == nil || !strings.Contains(err.Error(), "cards") {
		t.Errorf("Validate = %v, want an error that mentions cards", err)
	}
}
