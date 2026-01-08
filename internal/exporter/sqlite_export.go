// Package exporter provides functionality to export command sets from the database.
package exporter

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	// _ import for sqlite driver registration
	_ "modernc.org/sqlite"

	"github.com/VoxDroid/krnr/internal/config"
	dbpkg "github.com/VoxDroid/krnr/internal/db"
)

// ExportDatabase copies the active krnr database to dstPath.
func ExportDatabase(dstPath string) error {
	src, err := config.DBPath()
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source db: %w", err)
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}
	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create dst db: %w", err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy db: %w", err)
	}
	return nil
}

// ExportCommandSet exports a single named command set into a standalone SQLite DB
// at dstPath. If the named set does not exist an error is returned.
func ExportCommandSet(srcDB *sql.DB, name string, dstPath string) error {
	// Query the command set
	row := srcDB.QueryRow("SELECT id, name, description, created_at, last_run FROM command_sets WHERE name = ?", name)
	var id int64
	var csName string
	var description sql.NullString
	var createdAt string
	var lastRun sql.NullString
	if err := row.Scan(&id, &csName, &description, &createdAt, &lastRun); err != nil {
		return fmt.Errorf("select command_set: %w", err)
	}

	rows, err := srcDB.Query("SELECT position, command FROM commands WHERE command_set_id = ? ORDER BY position ASC", id)
	if err != nil {
		return fmt.Errorf("select commands: %w", err)
	}
	defer func() { _ = rows.Close() }()

	cmds := []string{}
	for rows.Next() {
		var pos int
		var cmd string
		if err := rows.Scan(&pos, &cmd); err != nil {
			return err
		}
		cmds = append(cmds, cmd)
	}

	// Create destination DB and apply schema
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}
	dstDB, err := sql.Open("sqlite", dstPath)
	if err != nil {
		return fmt.Errorf("open dst db: %w", err)
	}
	defer func() { _ = dstDB.Close() }()

	if err := dbpkg.ApplyMigrations(dstDB); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Insert command set
	res, err := dstDB.Exec("INSERT INTO command_sets (name, description, created_at, last_run) VALUES (?, ?, ?, ?)", csName, description, createdAt, lastRun)
	if err != nil {
		return fmt.Errorf("insert command_set: %w", err)
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for i, c := range cmds {
		if _, err := dstDB.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", newID, i+1, c); err != nil {
			return fmt.Errorf("insert command: %w", err)
		}
	}
	return nil
}
