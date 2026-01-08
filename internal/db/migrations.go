package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	// _ import for sqlite driver registration
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// ApplyMigrations applies the embedded schema SQL to the database and
// performs lightweight post-creation migrations (adding new columns when needed).
func ApplyMigrations(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	// Ensure new columns exist on upgrades
	if err := ensureCommandSetColumns(db); err != nil {
		return err
	}
	return nil
}

// ensureCommandSetColumns checks for optional columns and adds them when missing.
func ensureCommandSetColumns(db *sql.DB) error {
	// check for author_name column
	rows, err := db.Query("PRAGMA table_info(command_sets)")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		cols[name] = true
	}
	if !cols["author_name"] {
		if _, err := db.Exec("ALTER TABLE command_sets ADD COLUMN author_name TEXT"); err != nil {
			return err
		}
	}
	if !cols["author_email"] {
		if _, err := db.Exec("ALTER TABLE command_sets ADD COLUMN author_email TEXT"); err != nil {
			return err
		}
	}
	return nil
}
