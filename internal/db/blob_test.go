package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRejectBlobNameInsert(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test_blob?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	// Try inserting a blob name via []byte param (driver binds as blob)
	if _, err := db.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", []byte{0xff, 0xfe}, "x"); err == nil {
		t.Fatalf("expected blob insert to be rejected by trigger")
	}
}
