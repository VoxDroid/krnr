package importer

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	// _ import for sqlite driver registration
	_ "modernc.org/sqlite"

	"github.com/VoxDroid/krnr/internal/config"
)

// ImportDatabase copies srcPath into the default database location. If overwrite
// is false and the destination exists, an error is returned.
func ImportDatabase(srcPath string, overwrite bool) error {
	dst, err := config.DBPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil && !overwrite {
		return errors.New("destination database exists; use overwrite=true to replace")
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy db: %w", err)
	}
	return nil
}

func ensureUniqueName(dst *sql.DB, orig string) (string, error) {
	name := orig
	si := 1
	for {
		var cnt int
		r := dst.QueryRow("SELECT count(*) FROM command_sets WHERE name = ?", name)
		if err := r.Scan(&cnt); err != nil {
			return "", err
		}
		if cnt == 0 {
			return name, nil
		}
		name = fmt.Sprintf("%s-import-%d", orig, si)
		si++
	}
}

// ImportCommandSet imports all command sets from srcPath into the active DB. If
// name collisions occur, the function appends a suffix to the imported name.
func ImportCommandSet(srcPath string) error {
	src, err := sql.Open("sqlite", srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer func() { _ = src.Close() }()

	dstPath, err := config.DBPath()
	if err != nil {
		return err
	}
	dst, err := sql.Open("sqlite", dstPath)
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer func() { _ = dst.Close() }()

	rows, err := src.Query("SELECT id, name, description, created_at, last_run FROM command_sets")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var name string
		var desc sql.NullString
		var created string
		var lastRun sql.NullString
		if err := rows.Scan(&id, &name, &desc, &created, &lastRun); err != nil {
			return err
		}
		uName, err := ensureUniqueName(dst, name)
		if err != nil {
			return err
		}
		newID, err := insertCommandSet(dst, uName, desc, created, lastRun)
		if err != nil {
			return err
		}
		// copy commands
		if err := copyCommands(src, id, dst, newID); err != nil {
			return err
		}
	}
	return nil
}

func insertCommandSet(dst *sql.DB, name string, desc sql.NullString, created string, lastRun sql.NullString) (int64, error) {
	res, err := dst.Exec("INSERT INTO command_sets (name, description, created_at, last_run) VALUES (?, ?, ?, ?)", name, desc, created, lastRun)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return newID, nil
}
