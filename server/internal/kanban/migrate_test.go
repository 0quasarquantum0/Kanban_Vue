package kanban

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite" // registers the "sqlite" driver used below
)

// openRaw opens an empty database file with no schema.
func openRaw(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "raw.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() }) // a failed close of a test database is not interesting
	return db
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

func TestMigrateAppliesFilesInOrderAndOnlyOnce(t *testing.T) {
	db := openRaw(t)
	files := fstest.MapFS{
		"migrations/002_add_b.sql": {Data: []byte(`CREATE TABLE b (id INTEGER); INSERT INTO a VALUES (1);`)},
		"migrations/001_add_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER);`)},
		"migrations/README.md":     {Data: []byte(`not a migration`)},
	}

	if err := migrate(context.Background(), db, files, "migrations"); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := migrate(context.Background(), db, files, "migrations"); err != nil {
		t.Fatalf("second run: %v", err)
	}

	if got, want := tableNames(t, db), []string{"a", "b", "schema_migrations"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v", got, want)
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM a`).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("table a has %d rows (err %v), want 1: migration 002 must run exactly once", rows, err)
	}
}

func TestMigratePicksUpOnlyNewFilesOnLaterRuns(t *testing.T) {
	db := openRaw(t)
	files := fstest.MapFS{"m/001_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER);`)}}
	if err := migrate(context.Background(), db, files, "m"); err != nil {
		t.Fatal(err)
	}

	files["m/002_b.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE b (id INTEGER);`)}
	if err := migrate(context.Background(), db, files, "m"); err != nil {
		t.Fatal(err)
	}

	if got, want := tableNames(t, db), []string{"a", "b", "schema_migrations"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v", got, want)
	}
}

func TestMigrateRollsBackAFailingFileAndStops(t *testing.T) {
	db := openRaw(t)
	files := fstest.MapFS{
		"m/001_ok.sql":   {Data: []byte(`CREATE TABLE a (id INTEGER);`)},
		"m/002_bad.sql":  {Data: []byte(`CREATE TABLE b (id INTEGER); THIS IS NOT SQL;`)},
		"m/003_late.sql": {Data: []byte(`CREATE TABLE c (id INTEGER);`)},
	}

	err := migrate(context.Background(), db, files, "m")

	if err == nil {
		t.Fatal("migrate succeeded, want an error")
	}
	if got, want := tableNames(t, db), []string{"a", "schema_migrations"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v (the failed file must leave nothing behind, and later files must not run)", got, want)
	}
}

func TestMigrateRejectsAFileWithoutANumber(t *testing.T) {
	db := openRaw(t)
	files := fstest.MapFS{"m/init.sql": {Data: []byte(`CREATE TABLE a (id INTEGER);`)}}

	if err := migrate(context.Background(), db, files, "m"); err == nil {
		t.Error("migrate succeeded for a file named init.sql, want an error about its name")
	}
}
