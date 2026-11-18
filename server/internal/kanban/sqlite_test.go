package kanban

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
)

// openTestSQLite opens a database file in a directory that the test removes afterwards.
func openTestSQLite(t *testing.T, path string) *SQLite {
	t.Helper()
	store, err := OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return store
}

// The same behaviour tests that Memory passes (see store_suite_test.go).
func TestSQLiteStore(t *testing.T) {
	testStore(t, func(t *testing.T) Store {
		return openTestSQLite(t, filepath.Join(t.TempDir(), "kanban.db"))
	})
}

func TestSQLiteKeepsItsDataWhenReopened(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "kanban.db")
	first, err := OpenSQLite(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureSeeded(ctx, first); err != nil {
		t.Fatal(err)
	}
	board, _, err := first.AddBoard(ctx, "", "Second")
	if err != nil {
		t.Fatal(err)
	}
	before, err := first.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := openTestSQLite(t, path)
	after, err := second.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if after.Version != before.Version || after.Version != 2 {
		t.Errorf("version after reopening = %d, before = %d, want 2", after.Version, before.Version)
	}
	if mustJSON(t, after) != mustJSON(t, before) {
		t.Errorf("state changed across a reopen:\n got %s\nwant %s", mustJSON(t, after), mustJSON(t, before))
	}
	if got := after.State.BoardIDs(); got[len(got)-1] != board.ID {
		t.Errorf("the new board should still be last: %v", got)
	}
	// EnsureSeeded on a reopened database must not seed a second time.
	if err := EnsureSeeded(ctx, second); err != nil {
		t.Fatal(err)
	}
	if got := len(mustSnapshot(t, second).State.Boards); got != 2 {
		t.Errorf("%d boards after EnsureSeeded on a used database, want 2", got)
	}
}

func mustSnapshot(t *testing.T, store Store) Snapshot {
	t.Helper()
	snap, err := store.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestSQLiteOpensTwiceWithoutReapplyingMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kanban.db")
	openTestSQLite(t, path)

	second, err := OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatalf("opening an already migrated database: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteEnforcesForeignKeysAndCascades(t *testing.T) {
	ctx := context.Background()
	store := openTestSQLite(t, filepath.Join(t.TempDir(), "kanban.db"))
	board, _, _ := store.AddBoard(ctx, "", "Board")
	column, _, _ := store.AddColumn(ctx, board.ID, "", "Column")
	if _, _, err := store.AddCard(ctx, column.ID, "", "Card"); err != nil {
		t.Fatal(err)
	}

	// A card that points at a column that does not exist is refused by the database itself.
	_, err := store.db.ExecContext(ctx,
		`INSERT INTO cards (id, column_id, title, description, labels, created_at, position)
		 VALUES ('x', 'no-such-column', 't', '', '[]', '', 0)`)
	if err == nil {
		t.Error("inserting a card with an unknown column_id succeeded; foreign keys are not enforced")
	}

	// Deleting a board in SQL, bypassing our code, still removes everything beneath it.
	if _, err := store.db.ExecContext(ctx, `DELETE FROM boards WHERE id = ?`, board.ID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"columns", "cards"} {
		var count int
		if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%d rows left in %s after deleting the board; ON DELETE CASCADE is missing", count, table)
		}
	}
}

func TestSQLiteKeepsPositionsDense(t *testing.T) {
	ctx := context.Background()
	store := openTestSQLite(t, filepath.Join(t.TempDir(), "kanban.db"))
	board, _, _ := store.AddBoard(ctx, "", "Board")
	a, _, _ := store.AddColumn(ctx, board.ID, "", "A")
	b, _, _ := store.AddColumn(ctx, board.ID, "", "B")
	var cards []string
	for _, title := range []string{"1", "2", "3", "4"} {
		card, _, err := store.AddCard(ctx, a.ID, "", title)
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card.ID)
	}

	for _, step := range []func() error{
		func() error { _, err := store.MoveCard(ctx, cards[0], b.ID, 0); return err },
		func() error { _, err := store.MoveCard(ctx, cards[3], a.ID, 0); return err },
		func() error { _, err := store.MoveCard(ctx, cards[1], b.ID, 99); return err },
		func() error { _, err := store.DeleteCard(ctx, cards[2]); return err },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
		assertDensePositions(t, store.db, "cards", "column_id")
	}
	if _, err := store.DeleteColumn(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	assertDensePositions(t, store.db, "columns", "board_id")
}

// assertDensePositions checks that, within each parent, positions are exactly 0, 1, 2, …
func assertDensePositions(t *testing.T, db *sql.DB, table, parent string) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT `+parent+`, position FROM `+table+` ORDER BY `+parent+`, position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	positions := map[string][]int{}
	for rows.Next() {
		var owner string
		var position int
		if err := rows.Scan(&owner, &position); err != nil {
			t.Fatal(err)
		}
		positions[owner] = append(positions[owner], position)
	}
	for owner, got := range positions {
		want := make([]int, len(got))
		for i := range want {
			want[i] = i
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s positions under %s %q = %v, want %v", table, parent, owner, got, want)
		}
	}
}
