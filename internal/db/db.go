package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/drei/krnr/internal/config"
)

// InitDB ensures the data directory exists, opens the SQLite database, and
// creates the schema if it does not exist.
func InitDB() (*sql.DB, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	schema := `CREATE TABLE IF NOT EXISTS command_sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    last_run DATETIME
);

CREATE TABLE IF NOT EXISTS commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_set_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    command TEXT NOT NULL,
    FOREIGN KEY(command_set_id) REFERENCES command_sets(id)
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS command_set_tags (
    command_set_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (command_set_id, tag_id)
);
`)

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
