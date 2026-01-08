package db

import (
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/config"
)

func TestInitDBCreatesFileAndSchema(t *testing.T) {
	tmp := t.TempDir()
	// Ensure user home resolves to tmp for DBPath
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	dbPath, err := config.DBPath()
	if err != nil {
		t.Fatalf("DBPath(): %v", err)
	}

	// remove any existing file
	_ = os.Remove(dbPath)

	db, err := InitDB()
	if err != nil {
		t.Fatalf("InitDB() error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db file not created: %v", err)
	}

	var count int
	r := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='command_sets'")
	if err := r.Scan(&count); err != nil {
		t.Fatalf("query schema: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected table 'command_sets' to exist")
	}

	// Basic smoke test: ensure we can insert a command_set
	res, err := db.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", "testset", "desc")
	if err != nil {
		t.Fatalf("insert command_set failed: %v", err)
	}
	_ = res
}
