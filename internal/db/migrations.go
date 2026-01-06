package db

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"embed"
)

//go:embed schema.sql
var schemaSQL string

// ApplyMigrations applies the embedded schema SQL to the database.
func ApplyMigrations(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
