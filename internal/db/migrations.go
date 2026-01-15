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

	// Validate existing data integrity: no empty trimmed names and no duplicate trimmed names.
	// This prevents silent acceptance of invalid rows from older DBs or imports.
	var cnt int
	row := db.QueryRow("SELECT count(*) FROM command_sets WHERE trim(name) = '' OR name IS NULL")
	if err := row.Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("apply migrations: found %d command_sets with empty trimmed names; please remove or fix them before starting", cnt)
	}

	rows, err := db.Query("SELECT TRIM(name) as tname, count(*) as c FROM command_sets GROUP BY TRIM(name) HAVING c > 1")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var dupes []string
	for rows.Next() {
		var t string
		var c int
		if err := rows.Scan(&t, &c); err != nil {
			return err
		}
		dupes = append(dupes, t)
	}
	if len(dupes) > 0 {
		return fmt.Errorf("apply migrations: found duplicate command_set names (trimmed): %v; please dedupe before starting", dupes)
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
