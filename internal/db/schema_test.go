package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTriggersRejectEmptyAndDuplicateInserts(t *testing.T) {
	// in-memory DB
	db, err := sql.Open("sqlite", "file:test_triggers?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := ApplyMigrations(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	// empty name insert should fail
	if _, err := db.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", "   ", "x"); err == nil {
		t.Fatalf("expected insert with empty name to be rejected by trigger")
	}

	// good insert should succeed
	if _, err := db.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", "valid", "x"); err != nil {
		t.Fatalf("unexpected insert error: %v", err)
	}

	// duplicate (trimmed) insert should fail
	if _, err := db.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", " valid ", "x"); err == nil {
		t.Fatalf("expected duplicate trimmed insert to be rejected by trigger")
	}
}
