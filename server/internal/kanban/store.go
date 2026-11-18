package kanban

import (
	"context"
	"encoding/json"
	"strings"
)

// Snapshot is the whole state together with the version it was read at.
type Snapshot struct {
	Version int64 `json:"version"`
	State   State `json:"state"`
}

// Optional is a JSON field that can be absent, null, or have a value. A plain pointer cannot
// tell "absent" (leave it alone) from "null" (clear it); Set can.
type Optional[T any] struct {
	Value T
	Set   bool
}

// UnmarshalJSON is only called when the key is present in the JSON, including when its value
// is null, so reaching it at all is what marks the field as set.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	return json.Unmarshal(data, &o.Value)
}

// CardPatch is the editable part of a card. A nil field (or an unset DueDate) is left as it
// is. It mirrors CardPatch in src/types.ts, where a missing key means "unchanged".
type CardPatch struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Labels      *[]string         `json:"labels"`
	DueDate     Optional[*string] `json:"dueDate"` // set to nil to clear the due date
}

// Store keeps the Kanban state. Every method that changes something returns the new version
// of the state. IDs passed to the Add methods are requests: see ChooseID.
//
// Operations on something that does not exist return an error matching ErrNotFound, and bad
// input returns one matching ErrInvalid. Deleting a board deletes its columns and their cards;
// deleting a column deletes its cards.
type Store interface {
	// Snapshot returns a copy of the state and its version.
	Snapshot(ctx context.Context) (Snapshot, error)
	// ReplaceState swaps in a whole new state, but only if baseVersion is still the current
	// version; otherwise it returns ErrConflict.
	ReplaceState(ctx context.Context, baseVersion int64, state State) (version int64, err error)

	AddBoard(ctx context.Context, id, title string) (Board, int64, error)
	RenameBoard(ctx context.Context, boardID, title string) (int64, error)
	DeleteBoard(ctx context.Context, boardID string) (int64, error)

	AddColumn(ctx context.Context, boardID, id, title string) (Column, int64, error)
	RenameColumn(ctx context.Context, columnID, title string) (int64, error)
	DeleteColumn(ctx context.Context, columnID string) (int64, error)

	AddCard(ctx context.Context, columnID, id, title string) (Card, int64, error)
	UpdateCard(ctx context.Context, cardID string, patch CardPatch) (Card, int64, error)
	DeleteCard(ctx context.Context, cardID string) (int64, error)
	// MoveCard puts the card at toIndex in the target column, which may be the one it is
	// already in. An index outside the column is clamped, so a large one appends.
	MoveCard(ctx context.Context, cardID, toColumnID string, toIndex int) (int64, error)
}

// EnsureSeeded fills an untouched store (version 0) with the demo board. A store that has been
// written to, even if the user has since deleted everything, is left alone.
func EnsureSeeded(ctx context.Context, store Store) error {
	snap, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}
	if snap.Version != 0 {
		return nil
	}
	_, err = store.ReplaceState(ctx, 0, NewSeedState())
	return err
}

// checkTitle is the one rule for the names of boards, columns and cards.
func checkTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return invalidf("title must not be empty")
	}
	return nil
}

// Clone returns a deep copy of the state that shares no maps or slices with the original.
func (s State) Clone() State {
	c := State{
		Boards:     make(map[string]Board, len(s.Boards)),
		Columns:    make(map[string]Column, len(s.Columns)),
		Cards:      make(map[string]Card, len(s.Cards)),
		BoardOrder: append([]string(nil), s.BoardOrder...),
	}
	for id, board := range s.Boards {
		board.ColumnIDs = cloneIDs(board.ColumnIDs)
		c.Boards[id] = board
	}
	for id, column := range s.Columns {
		column.CardIDs = cloneIDs(column.CardIDs)
		c.Columns[id] = column
	}
	for id, card := range s.Cards {
		c.Cards[id] = cloneCard(card)
	}
	return c
}

// cloneIDs copies a slice, keeping nil as nil and empty as empty (the difference matters to
// Validate and to the JSON encoding).
func cloneIDs(ids []string) []string {
	if ids == nil {
		return nil
	}
	return append([]string{}, ids...)
}

func cloneCard(card Card) Card {
	card.Labels = cloneIDs(card.Labels)
	if card.DueDate != nil {
		due := *card.DueDate
		card.DueDate = &due
	}
	return card
}

// apply changes the card as the patch describes.
func (p CardPatch) apply(card Card) (Card, error) {
	if p.Title != nil {
		if err := checkTitle(*p.Title); err != nil {
			return Card{}, err
		}
		card.Title = *p.Title
	}
	if p.Description != nil {
		card.Description = *p.Description
	}
	if p.Labels != nil {
		card.Labels = append([]string{}, *p.Labels...)
	}
	if p.DueDate.Set {
		card.DueDate = p.DueDate.Value
	}
	return cloneCard(card), nil
}
