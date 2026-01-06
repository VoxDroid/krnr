package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/VoxDroid/krnr/internal/config"
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

	if err := ApplyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
