package importer

import (
	"database/sql"
	"testing"

	dbpkg "github.com/VoxDroid/krnr/internal/db"
	_ "modernc.org/sqlite"
)

func TestInsertCommandSetRejectsEmptyName(t *testing.T) {
	dbPath := "file:test_insert_empty?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// apply migrations so table exists
	if err := dbpkg.ApplyMigrations(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	if _, err := insertCommandSet(db, "   ", sql.NullString{Valid: false}, "2020-01-01T00:00:00Z", sql.NullString{Valid: false}); err == nil {
		t.Fatalf("expected error when inserting empty name, got nil")
	}
}

func TestApplyMigrationsDetectsDuplicateTrimmedNames(t *testing.T) {
	// create a DB with duplicate trimmed names before migrations run
	dbPath := "file:test_dup_detect?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// create a minimal command_sets table (older DB state)
	if _, err := db.Exec(`CREATE TABLE command_sets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO command_sets (name, description, created_at) VALUES (' a ', 'd', datetime('now'))`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO command_sets (name, description, created_at) VALUES ('a', 'd2', datetime('now'))`); err != nil {
		t.Fatalf("insert dup: %v", err)
	}

	// now apply migrations; this should detect the duplicate trimmed names and return an error
	if err := dbpkg.ApplyMigrations(db); err == nil {
		t.Fatalf("expected migrations to detect duplicate trimmed names, but apply succeeded")
	}
}
