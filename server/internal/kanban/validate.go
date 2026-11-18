package kanban

import (
	"maps"
	"slices"
)

// Validate checks that a state is well formed and returns an error saying what is wrong, or
// nil. It ports isKanbanState() from src/validation.ts and adds two rules the database needs:
//   - every entity is stored under its own ID;
//   - every column is listed by a board, and every card by a column, exactly once. The
//     frontend only requires that references exist; a relational database cannot store a
//     column with no board, so the server is stricter.
//
// Fields that are missing or null in the JSON arrive here as nil maps and slices, which is how
// "labels is required" can be told apart from an empty list: [] decodes to a non-nil slice.
func Validate(s State) error {
	if s.Boards == nil {
		return invalidf("boards is required")
	}
	if s.Columns == nil {
		return invalidf("columns is required")
	}
	if s.Cards == nil {
		return invalidf("cards is required")
	}

	// Sorted keys make the first reported error the same every time.
	for _, key := range slices.Sorted(maps.Keys(s.Boards)) {
		board := s.Boards[key]
		if board.ID != key {
			return invalidf("board %q is stored under the key %q", board.ID, key)
		}
		if board.ColumnIDs == nil {
			return invalidf("board %q: columnIds is required", key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(s.Columns)) {
		column := s.Columns[key]
		if column.ID != key {
			return invalidf("column %q is stored under the key %q", column.ID, key)
		}
		if column.CardIDs == nil {
			return invalidf("column %q: cardIds is required", key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(s.Cards)) {
		card := s.Cards[key]
		if card.ID != key {
			return invalidf("card %q is stored under the key %q", card.ID, key)
		}
		if card.Labels == nil {
			return invalidf("card %q: labels is required", key)
		}
	}

	var columnRefs, cardRefs []string
	for _, key := range slices.Sorted(maps.Keys(s.Boards)) {
		columnRefs = append(columnRefs, s.Boards[key].ColumnIDs...)
	}
	for _, key := range slices.Sorted(maps.Keys(s.Columns)) {
		cardRefs = append(cardRefs, s.Columns[key].CardIDs...)
	}
	if err := checkReferences("column", columnRefs, s.Columns, "board"); err != nil {
		return err
	}
	return checkReferences("card", cardRefs, s.Cards, "column")
}

// checkReferences verifies that every ID in refs names an entity, that none is listed twice,
// and that every entity is listed. kind and owner only shape the error messages.
func checkReferences[V any](kind string, refs []string, entities map[string]V, owner string) error {
	listed := make(map[string]bool, len(refs))
	for _, id := range refs {
		if _, ok := entities[id]; !ok {
			return invalidf("a %s refers to the %s %q, which does not exist", owner, kind, id)
		}
		if listed[id] {
			return invalidf("the %s %q is listed more than once", kind, id)
		}
		listed[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(entities)) {
		if !listed[id] {
			return invalidf("the %s %q is not listed by any %s", kind, id, owner)
		}
	}
	return nil
}
