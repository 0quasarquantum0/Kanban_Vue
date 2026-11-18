package kanban

import (
	"slices"
	"testing"
)

func TestSeedStateMatchesTheDemoBoard(t *testing.T) {
	state := NewSeedState()

	if err := Validate(state); err != nil {
		t.Fatalf("the seed is not valid: %v", err)
	}
	if len(state.Boards) != 1 || len(state.Columns) != 3 || len(state.Cards) != 6 {
		t.Fatalf("got %d boards, %d columns, %d cards; want 1, 3, 6",
			len(state.Boards), len(state.Columns), len(state.Cards))
	}

	board := state.Boards[state.BoardIDs()[0]]
	if board.Title != "Vue Kanban" {
		t.Errorf("board title = %q, want %q", board.Title, "Vue Kanban")
	}
	var titles []string
	var cardCounts []int
	for _, columnID := range board.ColumnIDs {
		column := state.Columns[columnID]
		titles = append(titles, column.Title)
		cardCounts = append(cardCounts, len(column.CardIDs))
	}
	if want := []string{"To do", "Doing", "Done"}; !slices.Equal(titles, want) {
		t.Errorf("column titles = %v, want %v", titles, want)
	}
	if want := []int{3, 2, 1}; !slices.Equal(cardCounts, want) {
		t.Errorf("cards per column = %v, want %v", cardCounts, want)
	}
}

func TestSeedStateUsesNewIDsOnEveryCall(t *testing.T) {
	a, b := NewSeedState(), NewSeedState()

	for id := range a.Boards {
		if _, shared := b.Boards[id]; shared {
			t.Errorf("both seeds contain board %q", id)
		}
	}
}

func TestSeedCardsHaveLabelsAndATimestamp(t *testing.T) {
	for id, card := range NewSeedState().Cards {
		if card.Labels == nil {
			t.Errorf("card %q has nil labels; they must encode as [] not null", id)
		}
		if card.CreatedAt == "" {
			t.Errorf("card %q has no createdAt", id)
		}
	}
}
