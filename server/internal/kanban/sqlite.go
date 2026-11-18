package kanban

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite" // the blank import runs the driver's init, which registers "sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// SQLite is a Store backed by a SQLite database file.
type SQLite struct {
	db *sql.DB
}

// Compile-time check that *SQLite satisfies Store.
var _ Store = (*SQLite)(nil)

// OpenSQLite opens (creating it if needed) the database at path and brings its schema up to
// date by applying the embedded migrations.
func OpenSQLite(ctx context.Context, path string) (*SQLite, error) {
	// The pragmas are applied to every new connection: foreign keys are off by default in
	// SQLite, and without them ON DELETE CASCADE would silently do nothing.
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Set("_txlock", "immediate")  // take the write lock when a transaction begins
	dsn := path + "?" + query.Encode() // modernc reads the pragmas after the ? and strips them

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	// One connection serialises all access, which is plenty for a small app and means the
	// methods below never wait on each other's locks inside the database.
	db.SetMaxOpenConns(1)

	if err := migrate(ctx, db, migrationFiles, "migrations"); err != nil {
		_ = db.Close() // already failing; the migration error is the one to report
		return nil, err
	}
	return &SQLite{db: db}, nil
}

// Close releases the database file.
func (s *SQLite) Close() error { return s.db.Close() }

// transact runs fn inside a transaction. If fn succeeds, the version is bumped and everything
// is committed together; if it fails, nothing is kept.
func (s *SQLite) transact(ctx context.Context, fn func(tx *sql.Tx) error) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	// Rollback after a successful Commit is a harmless no-op (it returns sql.ErrTxDone), so
	// this one line covers every early return below.
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return 0, err
	}
	var version int64
	if err := tx.QueryRowContext(ctx, `UPDATE meta SET version = version + 1 RETURNING version`).Scan(&version); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return version, nil
}

// Snapshot implements Store.
func (s *SQLite) Snapshot(ctx context.Context) (Snapshot, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }() // read-only: there is nothing to commit

	snap := Snapshot{State: State{
		Boards:     map[string]Board{},
		Columns:    map[string]Column{},
		Cards:      map[string]Card{},
		BoardOrder: []string{},
	}}
	if err := tx.QueryRowContext(ctx, `SELECT version FROM meta`).Scan(&snap.Version); err != nil {
		return Snapshot{}, err
	}
	if err := loadBoards(ctx, tx, &snap.State); err != nil {
		return Snapshot{}, err
	}
	if err := loadColumns(ctx, tx, &snap.State); err != nil {
		return Snapshot{}, err
	}
	if err := loadCards(ctx, tx, &snap.State); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func loadBoards(ctx context.Context, tx *sql.Tx, state *State) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, title FROM boards ORDER BY position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		board := Board{ColumnIDs: []string{}}
		if err := rows.Scan(&board.ID, &board.Title); err != nil {
			return err
		}
		state.Boards[board.ID] = board
		state.BoardOrder = append(state.BoardOrder, board.ID)
	}
	return rows.Err()
}

func loadColumns(ctx context.Context, tx *sql.Tx, state *State) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, board_id, title FROM columns ORDER BY position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		column := Column{CardIDs: []string{}}
		var boardID string
		if err := rows.Scan(&column.ID, &boardID, &column.Title); err != nil {
			return err
		}
		state.Columns[column.ID] = column
		board := state.Boards[boardID]
		board.ColumnIDs = append(board.ColumnIDs, column.ID)
		state.Boards[boardID] = board
	}
	return rows.Err()
}

func loadCards(ctx context.Context, tx *sql.Tx, state *State) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, column_id, title, description, labels, due_date, created_at FROM cards ORDER BY position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			card     Card
			columnID string
			labels   string
			dueDate  sql.NullString
		)
		if err := rows.Scan(&card.ID, &columnID, &card.Title, &card.Description, &labels, &dueDate, &card.CreatedAt); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(labels), &card.Labels); err != nil {
			return fmt.Errorf("card %q has unreadable labels: %w", card.ID, err)
		}
		if card.Labels == nil {
			card.Labels = []string{}
		}
		if dueDate.Valid {
			card.DueDate = &dueDate.String
		}
		state.Cards[card.ID] = card
		column := state.Columns[columnID]
		column.CardIDs = append(column.CardIDs, card.ID)
		state.Columns[columnID] = column
	}
	return rows.Err()
}

// ReplaceState implements Store.
func (s *SQLite) ReplaceState(ctx context.Context, baseVersion int64, state State) (int64, error) {
	if err := Validate(state); err != nil {
		return 0, err
	}
	return s.transact(ctx, func(tx *sql.Tx) error {
		var current int64
		if err := tx.QueryRowContext(ctx, `SELECT version FROM meta`).Scan(&current); err != nil {
			return err
		}
		if current != baseVersion {
			return fmt.Errorf("%w: the state is at version %d, not %d", ErrConflict, current, baseVersion)
		}

		for _, table := range []string{"cards", "columns", "boards"} { // children first
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table); err != nil {
				return err
			}
		}
		for position, boardID := range state.BoardIDs() {
			board := state.Boards[boardID]
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO boards (id, title, position) VALUES (?, ?, ?)`, board.ID, board.Title, position); err != nil {
				return err
			}
			for columnPosition, columnID := range board.ColumnIDs {
				column := state.Columns[columnID]
				if _, err := tx.ExecContext(ctx,
					`INSERT INTO columns (id, board_id, title, position) VALUES (?, ?, ?, ?)`,
					column.ID, board.ID, column.Title, columnPosition); err != nil {
					return err
				}
				for cardPosition, cardID := range column.CardIDs {
					if err := insertCard(ctx, tx, state.Cards[cardID], column.ID, cardPosition); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func insertCard(ctx context.Context, tx *sql.Tx, card Card, columnID string, position int) error {
	labels, err := json.Marshal(card.Labels)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO cards (id, column_id, title, description, labels, due_date, created_at, position)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		card.ID, columnID, card.Title, card.Description, string(labels), card.DueDate, card.CreatedAt, position)
	return err
}

// exists reports whether a row with this id is in the table. The table name is only ever one
// of this file's own constants, never user input (SQL cannot take a table name as a ? value).
func exists(ctx context.Context, tx *sql.Tx, table, id string) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE id = ?`, id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// freeID picks the ID for a new row in table, as ChooseID does, by asking the database.
func freeID(ctx context.Context, tx *sql.Tx, table, requested string) (string, error) {
	var lookupErr error
	id := ChooseID(requested, func(id string) bool {
		used, err := exists(ctx, tx, table, id)
		if err != nil {
			lookupErr = err // remember it, and stop the search by claiming the ID is free
		}
		return used
	})
	return id, lookupErr
}

// nextPosition is the position just after the last row of table with the given parent.
func nextPosition(ctx context.Context, tx *sql.Tx, query string, args ...any) (int, error) {
	var position int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&position)
	return position, err
}

// AddBoard implements Store.
func (s *SQLite) AddBoard(ctx context.Context, id, title string) (Board, int64, error) {
	if err := checkTitle(title); err != nil {
		return Board{}, 0, err
	}
	board := Board{Title: title, ColumnIDs: []string{}}
	version, err := s.transact(ctx, func(tx *sql.Tx) error {
		var err error
		if board.ID, err = freeID(ctx, tx, "boards", id); err != nil {
			return err
		}
		position, err := nextPosition(ctx, tx, `SELECT COALESCE(MAX(position) + 1, 0) FROM boards`)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO boards (id, title, position) VALUES (?, ?, ?)`, board.ID, title, position)
		return err
	})
	return board, version, err
}

// rename sets the title of one row, or reports ErrNotFound.
func (s *SQLite) rename(ctx context.Context, table, kind, id, title string) (int64, error) {
	if err := checkTitle(title); err != nil {
		return 0, err
	}
	return s.transact(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE `+table+` SET title = ? WHERE id = ?`, title, id)
		if err != nil {
			return err
		}
		if changed, err := result.RowsAffected(); err != nil {
			return err
		} else if changed == 0 {
			return notFound(kind, id)
		}
		return nil
	})
}

// RenameBoard implements Store.
func (s *SQLite) RenameBoard(ctx context.Context, boardID, title string) (int64, error) {
	return s.rename(ctx, "boards", "board", boardID, title)
}

// RenameColumn implements Store.
func (s *SQLite) RenameColumn(ctx context.Context, columnID, title string) (int64, error) {
	return s.rename(ctx, "columns", "column", columnID, title)
}

// remove deletes one row (its children go with it through ON DELETE CASCADE) and closes the
// gap it leaves in the positions of its siblings. parentColumn is "" for boards, which have
// no parent.
func (s *SQLite) remove(ctx context.Context, table, kind, id, parentColumn string) (int64, error) {
	return s.transact(ctx, func(tx *sql.Tx) error {
		var position int
		var parent string
		selectParent := "'' "
		if parentColumn != "" {
			selectParent = parentColumn
		}
		err := tx.QueryRowContext(ctx,
			`SELECT position, `+selectParent+` FROM `+table+` WHERE id = ?`, id).Scan(&position, &parent)
		if errors.Is(err, sql.ErrNoRows) {
			return notFound(kind, id)
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
			return err
		}
		if parentColumn == "" {
			_, err = tx.ExecContext(ctx, `UPDATE `+table+` SET position = position - 1 WHERE position > ?`, position)
		} else {
			_, err = tx.ExecContext(ctx,
				`UPDATE `+table+` SET position = position - 1 WHERE `+parentColumn+` = ? AND position > ?`, parent, position)
		}
		return err
	})
}

// DeleteBoard implements Store.
func (s *SQLite) DeleteBoard(ctx context.Context, boardID string) (int64, error) {
	return s.remove(ctx, "boards", "board", boardID, "")
}

// DeleteColumn implements Store.
func (s *SQLite) DeleteColumn(ctx context.Context, columnID string) (int64, error) {
	return s.remove(ctx, "columns", "column", columnID, "board_id")
}

// DeleteCard implements Store.
func (s *SQLite) DeleteCard(ctx context.Context, cardID string) (int64, error) {
	return s.remove(ctx, "cards", "card", cardID, "column_id")
}

// AddColumn implements Store.
func (s *SQLite) AddColumn(ctx context.Context, boardID, id, title string) (Column, int64, error) {
	if err := checkTitle(title); err != nil {
		return Column{}, 0, err
	}
	column := Column{Title: title, CardIDs: []string{}}
	version, err := s.transact(ctx, func(tx *sql.Tx) error {
		if ok, err := exists(ctx, tx, "boards", boardID); err != nil {
			return err
		} else if !ok {
			return notFound("board", boardID)
		}
		var err error
		if column.ID, err = freeID(ctx, tx, "columns", id); err != nil {
			return err
		}
		position, err := nextPosition(ctx, tx,
			`SELECT COALESCE(MAX(position) + 1, 0) FROM columns WHERE board_id = ?`, boardID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO columns (id, board_id, title, position) VALUES (?, ?, ?, ?)`, column.ID, boardID, title, position)
		return err
	})
	return column, version, err
}

// AddCard implements Store.
func (s *SQLite) AddCard(ctx context.Context, columnID, id, title string) (Card, int64, error) {
	if err := checkTitle(title); err != nil {
		return Card{}, 0, err
	}
	card := Card{Title: title, Labels: []string{}, CreatedAt: Now()}
	version, err := s.transact(ctx, func(tx *sql.Tx) error {
		if ok, err := exists(ctx, tx, "columns", columnID); err != nil {
			return err
		} else if !ok {
			return notFound("column", columnID)
		}
		var err error
		if card.ID, err = freeID(ctx, tx, "cards", id); err != nil {
			return err
		}
		position, err := nextPosition(ctx, tx,
			`SELECT COALESCE(MAX(position) + 1, 0) FROM cards WHERE column_id = ?`, columnID)
		if err != nil {
			return err
		}
		return insertCard(ctx, tx, card, columnID, position)
	})
	return card, version, err
}

// UpdateCard implements Store.
func (s *SQLite) UpdateCard(ctx context.Context, cardID string, patch CardPatch) (Card, int64, error) {
	var updated Card
	version, err := s.transact(ctx, func(tx *sql.Tx) error {
		var (
			card    Card
			labels  string
			dueDate sql.NullString
		)
		err := tx.QueryRowContext(ctx,
			`SELECT id, title, description, labels, due_date, created_at FROM cards WHERE id = ?`, cardID).
			Scan(&card.ID, &card.Title, &card.Description, &labels, &dueDate, &card.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return notFound("card", cardID)
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(labels), &card.Labels); err != nil {
			return err
		}
		if dueDate.Valid {
			card.DueDate = &dueDate.String
		}

		if updated, err = patch.apply(card); err != nil {
			return err
		}
		newLabels, err := json.Marshal(updated.Labels)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE cards SET title = ?, description = ?, labels = ?, due_date = ? WHERE id = ?`,
			updated.Title, updated.Description, string(newLabels), updated.DueDate, cardID)
		return err
	})
	if err != nil {
		return Card{}, 0, err
	}
	return updated, version, nil
}

// MoveCard implements Store. The whole move, closing the gap it leaves and opening the one it
// needs, is a single transaction, so no reader ever sees the card missing or twice.
func (s *SQLite) MoveCard(ctx context.Context, cardID, toColumnID string, toIndex int) (int64, error) {
	return s.transact(ctx, func(tx *sql.Tx) error {
		var fromColumnID string
		var fromPosition int
		err := tx.QueryRowContext(ctx, `SELECT column_id, position FROM cards WHERE id = ?`, cardID).
			Scan(&fromColumnID, &fromPosition)
		if errors.Is(err, sql.ErrNoRows) {
			return notFound("card", cardID)
		}
		if err != nil {
			return err
		}
		if ok, err := exists(ctx, tx, "columns", toColumnID); err != nil {
			return err
		} else if !ok {
			return notFound("column", toColumnID)
		}

		// 1. Close the gap in the source column.
		if _, err := tx.ExecContext(ctx,
			`UPDATE cards SET position = position - 1 WHERE column_id = ? AND position > ?`,
			fromColumnID, fromPosition); err != nil {
			return err
		}
		// 2. Clamp the index to the target column without the moving card, then open a gap.
		var others int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM cards WHERE column_id = ? AND id <> ?`, toColumnID, cardID).Scan(&others); err != nil {
			return err
		}
		index := max(0, min(toIndex, others))
		if _, err := tx.ExecContext(ctx,
			`UPDATE cards SET position = position + 1 WHERE column_id = ? AND position >= ? AND id <> ?`,
			toColumnID, index, cardID); err != nil {
			return err
		}
		// 3. Put the card in the gap.
		_, err = tx.ExecContext(ctx, `UPDATE cards SET column_id = ?, position = ? WHERE id = ?`, toColumnID, index, cardID)
		return err
	})
}
