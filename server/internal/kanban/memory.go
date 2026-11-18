package kanban

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// Memory is a Store that keeps everything in memory and loses it on exit.
//
// Every method holds mu while it works: HTTP handlers run on many goroutines at once, and Go
// maps are not safe for concurrent use (two writes at the same moment crash the program).
// Browser JavaScript never had this problem because it runs on a single thread.
type Memory struct {
	mu      sync.Mutex
	state   State
	version int64
}

// Compile-time check that *Memory satisfies Store: it fails to build if a method is missing.
var _ Store = (*Memory)(nil)

// NewMemory returns an empty store at version 0.
func NewMemory() *Memory {
	return &Memory{state: State{
		Boards:     map[string]Board{},
		Columns:    map[string]Column{},
		Cards:      map[string]Card{},
		BoardOrder: []string{},
	}}
}

// bump records that the state changed. The caller must hold m.mu.
func (m *Memory) bump() int64 {
	m.version++
	return m.version
}

// Snapshot implements Store.
func (m *Memory) Snapshot(context.Context) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Snapshot{Version: m.version, State: m.state.Clone()}, nil
}

// ReplaceState implements Store.
func (m *Memory) ReplaceState(_ context.Context, baseVersion int64, state State) (int64, error) {
	if err := Validate(state); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if baseVersion != m.version {
		return 0, fmt.Errorf("%w: the state is at version %d, not %d", ErrConflict, m.version, baseVersion)
	}
	next := state.Clone()
	next.BoardOrder = next.BoardIDs()
	m.state = next
	return m.bump(), nil
}

// AddBoard implements Store.
func (m *Memory) AddBoard(_ context.Context, id, title string) (Board, int64, error) {
	if err := checkTitle(title); err != nil {
		return Board{}, 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	id = ChooseID(id, func(id string) bool { _, used := m.state.Boards[id]; return used })
	board := Board{ID: id, Title: title, ColumnIDs: []string{}}
	m.state.Boards[id] = board
	m.state.BoardOrder = append(m.state.BoardOrder, id)
	return board, m.bump(), nil
}

// RenameBoard implements Store.
func (m *Memory) RenameBoard(_ context.Context, boardID, title string) (int64, error) {
	if err := checkTitle(title); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	board, ok := m.state.Boards[boardID]
	if !ok {
		return 0, notFound("board", boardID)
	}
	board.Title = title
	m.state.Boards[boardID] = board
	return m.bump(), nil
}

// DeleteBoard implements Store.
func (m *Memory) DeleteBoard(_ context.Context, boardID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	board, ok := m.state.Boards[boardID]
	if !ok {
		return 0, notFound("board", boardID)
	}
	// Copy first: removeColumn edits the board's own columnIds slice.
	for _, columnID := range slices.Clone(board.ColumnIDs) {
		m.removeColumn(columnID)
	}
	delete(m.state.Boards, boardID)
	m.state.BoardOrder = slices.DeleteFunc(m.state.BoardOrder, func(id string) bool { return id == boardID })
	return m.bump(), nil
}

// AddColumn implements Store.
func (m *Memory) AddColumn(_ context.Context, boardID, id, title string) (Column, int64, error) {
	if err := checkTitle(title); err != nil {
		return Column{}, 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	board, ok := m.state.Boards[boardID]
	if !ok {
		return Column{}, 0, notFound("board", boardID)
	}
	id = ChooseID(id, func(id string) bool { _, used := m.state.Columns[id]; return used })
	column := Column{ID: id, Title: title, CardIDs: []string{}}
	m.state.Columns[id] = column
	board.ColumnIDs = append(board.ColumnIDs, id)
	m.state.Boards[boardID] = board
	return column, m.bump(), nil
}

// RenameColumn implements Store.
func (m *Memory) RenameColumn(_ context.Context, columnID, title string) (int64, error) {
	if err := checkTitle(title); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	column, ok := m.state.Columns[columnID]
	if !ok {
		return 0, notFound("column", columnID)
	}
	column.Title = title
	m.state.Columns[columnID] = column
	return m.bump(), nil
}

// DeleteColumn implements Store.
func (m *Memory) DeleteColumn(_ context.Context, columnID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.state.Columns[columnID]; !ok {
		return 0, notFound("column", columnID)
	}
	m.removeColumn(columnID)
	return m.bump(), nil
}

// removeColumn deletes a column, its cards, and its place on its board. The caller must hold
// m.mu and know that the column exists.
func (m *Memory) removeColumn(columnID string) {
	for _, cardID := range m.state.Columns[columnID].CardIDs {
		delete(m.state.Cards, cardID)
	}
	for id, board := range m.state.Boards {
		if i := slices.Index(board.ColumnIDs, columnID); i != -1 {
			board.ColumnIDs = slices.Delete(board.ColumnIDs, i, i+1)
			m.state.Boards[id] = board
		}
	}
	delete(m.state.Columns, columnID)
}

// AddCard implements Store.
func (m *Memory) AddCard(_ context.Context, columnID, id, title string) (Card, int64, error) {
	if err := checkTitle(title); err != nil {
		return Card{}, 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	column, ok := m.state.Columns[columnID]
	if !ok {
		return Card{}, 0, notFound("column", columnID)
	}
	id = ChooseID(id, func(id string) bool { _, used := m.state.Cards[id]; return used })
	card := Card{ID: id, Title: title, Labels: []string{}, CreatedAt: Now()}
	m.state.Cards[id] = card
	column.CardIDs = append(column.CardIDs, id)
	m.state.Columns[columnID] = column
	return cloneCard(card), m.bump(), nil
}

// UpdateCard implements Store.
func (m *Memory) UpdateCard(_ context.Context, cardID string, patch CardPatch) (Card, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	card, ok := m.state.Cards[cardID]
	if !ok {
		return Card{}, 0, notFound("card", cardID)
	}
	updated, err := patch.apply(card)
	if err != nil {
		return Card{}, 0, err
	}
	m.state.Cards[cardID] = updated
	return cloneCard(updated), m.bump(), nil
}

// DeleteCard implements Store.
func (m *Memory) DeleteCard(_ context.Context, cardID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.state.Cards[cardID]; !ok {
		return 0, notFound("card", cardID)
	}
	m.detachCard(cardID)
	delete(m.state.Cards, cardID)
	return m.bump(), nil
}

// detachCard removes the card's ID from whichever column lists it. The caller must hold m.mu.
func (m *Memory) detachCard(cardID string) {
	for id, column := range m.state.Columns {
		if i := slices.Index(column.CardIDs, cardID); i != -1 {
			column.CardIDs = slices.Delete(column.CardIDs, i, i+1)
			m.state.Columns[id] = column
			return
		}
	}
}

// MoveCard implements Store.
func (m *Memory) MoveCard(_ context.Context, cardID, toColumnID string, toIndex int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.state.Cards[cardID]; !ok {
		return 0, notFound("card", cardID)
	}
	if _, ok := m.state.Columns[toColumnID]; !ok {
		return 0, notFound("column", toColumnID)
	}

	m.detachCard(cardID)
	// Read the target after detaching: when moving within a column, detaching changed it.
	to := m.state.Columns[toColumnID]
	index := max(0, min(toIndex, len(to.CardIDs)))
	to.CardIDs = slices.Insert(to.CardIDs, index, cardID)
	m.state.Columns[toColumnID] = to
	return m.bump(), nil
}

// notFound builds an error that matches ErrNotFound and names what was missing.
func notFound(kind, id string) error {
	return fmt.Errorf("%s %q: %w", kind, id, ErrNotFound)
}
