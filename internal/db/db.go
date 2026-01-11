// Package db provides database utilities.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// _ import for sqlite driver registration
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

	// Set sensible pragmas to help cross-platform concurrency
	// - busy_timeout (ms): wait for a short period before returning SQLITE_BUSY
	// - journal_mode = WAL: improves concurrency for readers/writers
	_, _ = db.Exec("PRAGMA busy_timeout = 5000")
	_, _ = db.Exec("PRAGMA journal_mode = WAL")

	if err := ApplyMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
