package kanban

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

// migration is one numbered .sql file.
type migration struct {
	version int
	name    string
	file    string // path inside the fs.FS
}

// migrate applies every file in dir of fsys whose number has not been applied yet, in order.
// Files are named NNN_description.sql. Each runs in its own transaction together with the row
// that records it, so a file that fails leaves nothing behind and later files do not run.
func migrate(ctx context.Context, db *sql.DB, fsys fs.FS, dir string) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	available, err := listMigrations(fsys, dir)
	if err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range available {
		if applied[m.version] {
			continue
		}
		if err := applyMigration(ctx, db, fsys, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

func listMigrations(fsys fs.FS, dir string) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations: %w", err)
	}
	var list []migration
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
			continue
		}
		number, _, ok := strings.Cut(entry.Name(), "_")
		version, err := strconv.Atoi(number)
		if !ok || err != nil {
			return nil, fmt.Errorf("migration file %q: the name must look like 001_description.sql", entry.Name())
		}
		list = append(list, migration{version: version, name: entry.Name(), file: path.Join(dir, entry.Name())})
	}
	slices.SortFunc(list, func(a, b migration) int { return a.version - b.version })
	return list, nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("reading schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func applyMigration(ctx context.Context, db *sql.DB, fsys fs.FS, m migration) error {
	script, err := fs.ReadFile(fsys, m.file)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// After a successful Commit, Rollback returns sql.ErrTxDone and does nothing; after a
	// failure it undoes the whole file. Either way its own error is not worth reporting.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(script)); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.version, m.name, Now())
	if err != nil {
		return err
	}
	return tx.Commit()
}
