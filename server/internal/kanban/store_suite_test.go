package kanban

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
)

// testStore runs the behaviour every Store must have. Each implementation (the in-memory one,
// and the SQLite one in step 5) calls it with a function that makes a fresh, empty store, so
// both are held to the same tests. It ports src/stores/__tests__/kanban.spec.ts.
func testStore(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()

	// run starts a subtest with a fresh store and a small fixture around it.
	run := func(name string, test func(t *testing.T, f *fixture)) {
		t.Run(name, func(t *testing.T) {
			test(t, &fixture{t: t, ctx: context.Background(), store: newStore(t)})
		})
	}

	run("a new store is empty and at version 0", func(t *testing.T, f *fixture) {
		snap := f.snapshot()

		if snap.Version != 0 {
			t.Errorf("version = %d, want 0", snap.Version)
		}
		if len(snap.State.Boards)+len(snap.State.Columns)+len(snap.State.Cards) != 0 {
			t.Errorf("state = %+v, want it empty", snap.State)
		}
		if err := Validate(snap.State); err != nil {
			t.Errorf("an empty store's state is not valid: %v", err)
		}
	})

	run("Snapshot returns a copy that the caller can change freely", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		snap := f.snapshot()

		snap.State.Boards[board.ID] = Board{ID: board.ID, Title: "changed", ColumnIDs: []string{}}
		delete(snap.State.Boards, board.ID)

		if got := f.snapshot().State.Boards[board.ID].Title; got != "Board" {
			t.Errorf("board title in the store = %q after changing a snapshot, want %q", got, "Board")
		}
	})

	run("every successful write raises the version by one and failures do not", func(t *testing.T, f *fixture) {
		board, v1, err := f.store.AddBoard(f.ctx, "", "Board")
		f.noError(err)
		_, v2, err := f.store.AddColumn(f.ctx, board.ID, "", "Column")
		f.noError(err)
		_, err = f.store.RenameBoard(f.ctx, "missing", "x")
		f.wantError(err, ErrNotFound)

		if v1 != 1 || v2 != 2 {
			t.Errorf("versions = %d, %d; want 1, 2", v1, v2)
		}
		if got := f.snapshot().Version; got != 2 {
			t.Errorf("current version = %d, want 2 (the failed write must not count)", got)
		}
	})

	run("AddBoard appends boards in the order they were added", func(t *testing.T, f *fixture) {
		a, b, c := f.addBoard("A"), f.addBoard("B"), f.addBoard("C")

		got := f.snapshot().State.BoardIDs()

		if want := []string{a.ID, b.ID, c.ID}; !slices.Equal(got, want) {
			t.Errorf("board order = %v, want %v", got, want)
		}
	})

	run("AddBoard keeps a valid unused ID from the client", func(t *testing.T, f *fixture) {
		const wanted = "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6d"

		board, _, err := f.store.AddBoard(f.ctx, wanted, "Board")

		f.noError(err)
		if board.ID != wanted {
			t.Errorf("board ID = %q, want the requested %q", board.ID, wanted)
		}
	})

	run("AddBoard makes a new ID when the requested one is invalid or taken", func(t *testing.T, f *fixture) {
		first := f.addBoard("First")

		for _, requested := range []string{"not-a-uuid", first.ID} {
			board, _, err := f.store.AddBoard(f.ctx, requested, "Other")
			f.noError(err)
			if board.ID == requested || !IsValidID(board.ID) {
				t.Errorf("AddBoard with ID %q gave %q, want a new valid ID", requested, board.ID)
			}
		}
		if got := len(f.snapshot().State.Boards); got != 3 {
			t.Errorf("%d boards, want 3", got)
		}
	})

	run("titles must not be empty", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		column := f.addColumn(board.ID, "Column")
		card := f.addCard(column.ID, "Card")

		_, _, err := f.store.AddBoard(f.ctx, "", "  ")
		f.wantError(err, ErrInvalid)
		_, _, err = f.store.AddColumn(f.ctx, board.ID, "", "")
		f.wantError(err, ErrInvalid)
		_, _, err = f.store.AddCard(f.ctx, column.ID, "", "\t")
		f.wantError(err, ErrInvalid)
		_, err = f.store.RenameBoard(f.ctx, board.ID, "")
		f.wantError(err, ErrInvalid)
		_, err = f.store.RenameColumn(f.ctx, column.ID, "")
		f.wantError(err, ErrInvalid)
		empty := ""
		_, _, err = f.store.UpdateCard(f.ctx, card.ID, CardPatch{Title: &empty})
		f.wantError(err, ErrInvalid)

		state := f.snapshot().State
		if len(state.Boards) != 1 || len(state.Columns) != 1 || len(state.Cards) != 1 {
			t.Errorf("rejected writes changed the state: %+v", state)
		}
		if state.Cards[card.ID].Title != "Card" {
			t.Errorf("card title = %q, want it unchanged", state.Cards[card.ID].Title)
		}
	})

	run("RenameBoard and RenameColumn change only the title", func(t *testing.T, f *fixture) {
		board := f.addBoard("Old")
		column := f.addColumn(board.ID, "Old")

		_, err := f.store.RenameBoard(f.ctx, board.ID, "New board")
		f.noError(err)
		_, err = f.store.RenameColumn(f.ctx, column.ID, "New column")
		f.noError(err)

		state := f.snapshot().State
		if got := state.Boards[board.ID]; got.Title != "New board" || !slices.Equal(got.ColumnIDs, []string{column.ID}) {
			t.Errorf("board = %+v", got)
		}
		if got := state.Columns[column.ID]; got.Title != "New column" {
			t.Errorf("column = %+v", got)
		}
	})

	run("AddColumn appends to the board and fails for an unknown board", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		a := f.addColumn(board.ID, "A")
		b := f.addColumn(board.ID, "B")

		if got, want := f.snapshot().State.Boards[board.ID].ColumnIDs, []string{a.ID, b.ID}; !slices.Equal(got, want) {
			t.Errorf("columnIds = %v, want %v", got, want)
		}

		_, _, err := f.store.AddColumn(f.ctx, "missing", "", "C")
		f.wantError(err, ErrNotFound)
		if got := len(f.snapshot().State.Columns); got != 2 {
			t.Errorf("%d columns after a failed AddColumn, want 2", got)
		}
	})

	run("AddCard appends a card with defaults to the column", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		f.addCard(column.ID, "first")
		second := f.addCard(column.ID, "second")

		state := f.snapshot().State

		if got, want := f.titles(column.ID), []string{"first", "second"}; !slices.Equal(got, want) {
			t.Errorf("titles = %v, want %v", got, want)
		}
		card := state.Cards[second.ID]
		if card.Title != "second" || card.Description != "" || card.DueDate != nil || card.CreatedAt == "" {
			t.Errorf("card = %+v, want a title, no description, no due date and a creation time", card)
		}
		if card.Labels == nil || len(card.Labels) != 0 {
			t.Errorf("labels = %#v, want an empty non-nil slice", card.Labels)
		}
		if err := Validate(state); err != nil {
			t.Errorf("state is not valid after AddCard: %v", err)
		}
	})

	run("AddCard does nothing for an unknown column", func(t *testing.T, f *fixture) {
		f.addColumn(f.addBoard("Board").ID, "A")

		_, _, err := f.store.AddCard(f.ctx, "nope", "", "x")

		f.wantError(err, ErrNotFound)
		if got := len(f.snapshot().State.Cards); got != 0 {
			t.Errorf("%d cards, want 0", got)
		}
	})

	run("UpdateCard changes only the fields in the patch", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		card := f.addCard(column.ID, "title")
		description, due := "details", "2026-12-01"

		updated, _, err := f.store.UpdateCard(f.ctx, card.ID, CardPatch{
			Description: &description,
			Labels:      &[]string{"ui", "bug"},
			DueDate:     Optional[*string]{Value: &due, Set: true},
		})

		f.noError(err)
		got := f.snapshot().State.Cards[card.ID]
		if got.Title != "title" || got.Description != "details" || got.DueDate == nil || *got.DueDate != due ||
			!slices.Equal(got.Labels, []string{"ui", "bug"}) {
			t.Errorf("card = %+v", got)
		}
		if updated.Description != "details" {
			t.Errorf("the returned card = %+v, want the updated one", updated)
		}
	})

	run("UpdateCard can clear the due date and the labels", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		card := f.addCard(column.ID, "title")
		due := "2026-12-01"
		_, _, err := f.store.UpdateCard(f.ctx, card.ID, CardPatch{
			Labels:  &[]string{"x"},
			DueDate: Optional[*string]{Value: &due, Set: true},
		})
		f.noError(err)

		_, _, err = f.store.UpdateCard(f.ctx, card.ID, CardPatch{
			Labels:  &[]string{},
			DueDate: Optional[*string]{Set: true},
		})

		f.noError(err)
		got := f.snapshot().State.Cards[card.ID]
		if got.DueDate != nil {
			t.Errorf("due date = %v, want nil", *got.DueDate)
		}
		if got.Labels == nil || len(got.Labels) != 0 {
			t.Errorf("labels = %#v, want an empty non-nil slice", got.Labels)
		}
	})

	run("UpdateCard fails for an unknown card", func(t *testing.T, f *fixture) {
		title := "x"

		_, _, err := f.store.UpdateCard(f.ctx, "nope", CardPatch{Title: &title})

		f.wantError(err, ErrNotFound)
	})

	run("DeleteCard removes the card from its column", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		one, two := f.addCard(column.ID, "1"), f.addCard(column.ID, "2")

		_, err := f.store.DeleteCard(f.ctx, one.ID)

		f.noError(err)
		state := f.snapshot().State
		if _, ok := state.Cards[one.ID]; ok {
			t.Error("the card is still in the state")
		}
		if got, want := state.Columns[column.ID].CardIDs, []string{two.ID}; !slices.Equal(got, want) {
			t.Errorf("cardIds = %v, want %v", got, want)
		}
		_, err = f.store.DeleteCard(f.ctx, one.ID)
		f.wantError(err, ErrNotFound)
	})

	run("MoveCard reorders within a column", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		one := f.addCard(column.ID, "1")
		f.addCard(column.ID, "2")
		f.addCard(column.ID, "3")

		f.move(one.ID, column.ID, 2)
		if got, want := f.titles(column.ID), []string{"2", "3", "1"}; !slices.Equal(got, want) {
			t.Errorf("after moving to 2: %v, want %v", got, want)
		}

		f.move(one.ID, column.ID, 0)
		if got, want := f.titles(column.ID), []string{"1", "2", "3"}; !slices.Equal(got, want) {
			t.Errorf("after moving to 0: %v, want %v", got, want)
		}
	})

	run("MoveCard moves between columns at the given index", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		a, b := f.addColumn(board.ID, "A"), f.addColumn(board.ID, "B")
		one := f.addCard(a.ID, "1")
		f.addCard(a.ID, "2")
		f.addCard(b.ID, "x")
		f.addCard(b.ID, "y")

		f.move(one.ID, b.ID, 1)

		if got, want := f.titles(a.ID), []string{"2"}; !slices.Equal(got, want) {
			t.Errorf("column A = %v, want %v", got, want)
		}
		if got, want := f.titles(b.ID), []string{"x", "1", "y"}; !slices.Equal(got, want) {
			t.Errorf("column B = %v, want %v", got, want)
		}
		if err := Validate(f.snapshot().State); err != nil {
			t.Errorf("state is not valid after a move: %v", err)
		}
	})

	run("MoveCard clamps an out-of-range index", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		a, b := f.addColumn(board.ID, "A"), f.addColumn(board.ID, "B")
		one := f.addCard(a.ID, "1")
		f.addCard(b.ID, "x")

		f.move(one.ID, b.ID, 99)
		if got, want := f.titles(b.ID), []string{"x", "1"}; !slices.Equal(got, want) {
			t.Errorf("after moving to 99: %v, want %v", got, want)
		}

		f.move(one.ID, b.ID, -5)
		if got, want := f.titles(b.ID), []string{"1", "x"}; !slices.Equal(got, want) {
			t.Errorf("after moving to -5: %v, want %v", got, want)
		}
	})

	run("MoveCard leaves the card in place for an unknown target column or card", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		one := f.addCard(column.ID, "1")

		_, err := f.store.MoveCard(f.ctx, one.ID, "nope", 0)
		f.wantError(err, ErrNotFound)
		_, err = f.store.MoveCard(f.ctx, "nope", column.ID, 0)
		f.wantError(err, ErrNotFound)

		if got, want := f.titles(column.ID), []string{"1"}; !slices.Equal(got, want) {
			t.Errorf("titles = %v, want %v", got, want)
		}
	})

	run("DeleteColumn removes the column, its cards, and its place on the board", func(t *testing.T, f *fixture) {
		board := f.addBoard("Board")
		a, b := f.addColumn(board.ID, "A"), f.addColumn(board.ID, "B")
		one, two := f.addCard(a.ID, "1"), f.addCard(a.ID, "2")
		kept := f.addCard(b.ID, "kept")

		_, err := f.store.DeleteColumn(f.ctx, a.ID)

		f.noError(err)
		state := f.snapshot().State
		if _, ok := state.Columns[a.ID]; ok {
			t.Error("the column is still in the state")
		}
		if got, want := state.Boards[board.ID].ColumnIDs, []string{b.ID}; !slices.Equal(got, want) {
			t.Errorf("columnIds = %v, want %v", got, want)
		}
		for _, id := range []string{one.ID, two.ID} {
			if _, ok := state.Cards[id]; ok {
				t.Errorf("card %q survived its column", id)
			}
		}
		if _, ok := state.Cards[kept.ID]; !ok {
			t.Error("a card in another column was deleted")
		}
		_, err = f.store.DeleteColumn(f.ctx, a.ID)
		f.wantError(err, ErrNotFound)
	})

	run("DeleteBoard removes its columns and cards and leaves other boards alone", func(t *testing.T, f *fixture) {
		other := f.addBoard("Other")
		otherColumn := f.addColumn(other.ID, "Keep")
		otherCard := f.addCard(otherColumn.ID, "keep")
		board := f.addBoard("Doomed")
		a, b := f.addColumn(board.ID, "A"), f.addColumn(board.ID, "B")
		f.addCard(a.ID, "1")
		f.addCard(b.ID, "2")

		_, err := f.store.DeleteBoard(f.ctx, board.ID)

		f.noError(err)
		state := f.snapshot().State
		if got, want := state.BoardIDs(), []string{other.ID}; !slices.Equal(got, want) {
			t.Errorf("boards = %v, want %v", got, want)
		}
		if len(state.Columns) != 1 || len(state.Cards) != 1 {
			t.Errorf("%d columns and %d cards left, want 1 and 1", len(state.Columns), len(state.Cards))
		}
		if _, ok := state.Cards[otherCard.ID]; !ok {
			t.Error("the other board's card was deleted")
		}
		_, err = f.store.DeleteBoard(f.ctx, board.ID)
		f.wantError(err, ErrNotFound)
	})

	run("ReplaceState swaps in a valid state and keeps the order of the boards", func(t *testing.T, f *fixture) {
		f.addBoard("Old")
		next := NewSeedState()
		second := Board{ID: NewID(), Title: "Second", ColumnIDs: []string{}}
		next.Boards[second.ID] = second
		next.BoardOrder = []string{second.ID, next.BoardOrder[0]}

		version, err := f.store.ReplaceState(f.ctx, f.snapshot().Version, next)

		f.noError(err)
		snap := f.snapshot()
		if snap.Version != version || version != 2 {
			t.Errorf("version = %d (returned %d), want 2", snap.Version, version)
		}
		if got := snap.State.BoardIDs(); !slices.Equal(got, next.BoardOrder) {
			t.Errorf("board order = %v, want %v", got, next.BoardOrder)
		}
		if len(snap.State.Cards) != 6 {
			t.Errorf("%d cards, want the seed's 6", len(snap.State.Cards))
		}
		if wantJSON, gotJSON := mustJSON(t, next), mustJSON(t, snap.State); wantJSON != gotJSON {
			t.Errorf("state after ReplaceState differs:\n got %s\nwant %s", gotJSON, wantJSON)
		}
	})

	run("ReplaceState rejects a stale version with ErrConflict", func(t *testing.T, f *fixture) {
		f.addBoard("Board") // now at version 1

		_, err := f.store.ReplaceState(f.ctx, 0, NewSeedState())

		f.wantError(err, ErrConflict)
		if got := len(f.snapshot().State.Boards); got != 1 {
			t.Errorf("%d boards, want the original 1", got)
		}
	})

	run("ReplaceState rejects an invalid state and keeps the current one", func(t *testing.T, f *fixture) {
		f.addBoard("Board")
		broken := NewSeedState()
		for id, column := range broken.Columns {
			column.CardIDs = append(column.CardIDs, "missing")
			broken.Columns[id] = column
		}

		_, err := f.store.ReplaceState(f.ctx, 1, broken)

		f.wantError(err, ErrInvalid)
		snap := f.snapshot()
		if snap.Version != 1 || len(snap.State.Boards) != 1 {
			t.Errorf("version %d with %d boards, want 1 with 1", snap.Version, len(snap.State.Boards))
		}
	})

	run("EnsureSeeded fills a fresh store once", func(t *testing.T, f *fixture) {
		f.noError(EnsureSeeded(f.ctx, f.store))
		first := f.snapshot()
		f.noError(EnsureSeeded(f.ctx, f.store))
		second := f.snapshot()

		if len(first.State.Cards) != 6 {
			t.Errorf("%d cards after seeding, want 6", len(first.State.Cards))
		}
		if first.Version != second.Version {
			t.Errorf("seeding twice changed the version from %d to %d", first.Version, second.Version)
		}
	})

	run("EnsureSeeded leaves a store that was emptied by the user alone", func(t *testing.T, f *fixture) {
		board := f.addBoard("Only")
		_, err := f.store.DeleteBoard(f.ctx, board.ID)
		f.noError(err)

		f.noError(EnsureSeeded(f.ctx, f.store))

		if got := len(f.snapshot().State.Boards); got != 0 {
			t.Errorf("%d boards, want the user's empty state kept", got)
		}
	})

	run("concurrent writes are all applied", func(t *testing.T, f *fixture) {
		column := f.addColumn(f.addBoard("Board").ID, "A")
		const writers = 25

		var wg sync.WaitGroup
		for i := range writers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, _, err := f.store.AddCard(f.ctx, column.ID, "", fmt.Sprintf("card %d", i)); err != nil {
					t.Errorf("AddCard: %v", err)
				}
			}()
		}
		wg.Wait()

		snap := f.snapshot()
		if got := len(snap.State.Columns[column.ID].CardIDs); got != writers {
			t.Errorf("%d cards in the column, want %d", got, writers)
		}
		if want := int64(2 + writers); snap.Version != want {
			t.Errorf("version = %d, want %d", snap.Version, want)
		}
		if err := Validate(snap.State); err != nil {
			t.Errorf("state is not valid after concurrent writes: %v", err)
		}
	})
}

// fixture bundles a store with helpers that fail the test on unexpected errors, so the tests
// above can read as a list of steps.
type fixture struct {
	t     *testing.T
	ctx   context.Context
	store Store
}

func (f *fixture) noError(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatalf("unexpected error: %v", err)
	}
}

func (f *fixture) wantError(err, target error) {
	f.t.Helper()
	if !errors.Is(err, target) {
		f.t.Fatalf("error = %v, want one matching %v", err, target)
	}
}

func (f *fixture) snapshot() Snapshot {
	f.t.Helper()
	snap, err := f.store.Snapshot(f.ctx)
	f.noError(err)
	return snap
}

func (f *fixture) addBoard(title string) Board {
	f.t.Helper()
	board, _, err := f.store.AddBoard(f.ctx, "", title)
	f.noError(err)
	return board
}

func (f *fixture) addColumn(boardID, title string) Column {
	f.t.Helper()
	column, _, err := f.store.AddColumn(f.ctx, boardID, "", title)
	f.noError(err)
	return column
}

func (f *fixture) addCard(columnID, title string) Card {
	f.t.Helper()
	card, _, err := f.store.AddCard(f.ctx, columnID, "", title)
	f.noError(err)
	return card
}

func (f *fixture) move(cardID, columnID string, index int) {
	f.t.Helper()
	_, err := f.store.MoveCard(f.ctx, cardID, columnID, index)
	f.noError(err)
}

// titles lists the titles of a column's cards in order.
func (f *fixture) titles(columnID string) []string {
	f.t.Helper()
	state := f.snapshot().State
	titles := []string{}
	for _, id := range state.Columns[columnID].CardIDs {
		titles = append(titles, state.Cards[id].Title)
	}
	return titles
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
