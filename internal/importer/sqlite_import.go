package importer

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy db: %w", err)
	}
	return nil
}

// ImportCommandSet imports all command sets from srcPath into the active DB. If
// name collisions occur, the function appends a suffix to the imported name.
func ImportCommandSet(srcPath string) error {
	src, err := sql.Open("sqlite", srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()

	dstPath, err := config.DBPath()
	if err != nil {
		return err
	}
	dst, err := sql.Open("sqlite", dstPath)
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	rows, err := src.Query("SELECT id, name, description, created_at, last_run FROM command_sets")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		var desc sql.NullString
		var created string
		var lastRun sql.NullString
		if err := rows.Scan(&id, &name, &desc, &created, &lastRun); err != nil {
			return err
		}
		// Ensure unique name in dst
		orig := name
		si := 1
		for {
			var cnt int
			r := dst.QueryRow("SELECT count(*) FROM command_sets WHERE name = ?", name)
			if err := r.Scan(&cnt); err != nil {
				return err
			}
			if cnt == 0 {
				break
			}
			name = fmt.Sprintf("%s-import-%d", orig, si)
			si++
		}
		res, err := dst.Exec("INSERT INTO command_sets (name, description, created_at, last_run) VALUES (?, ?, ?, ?)", name, desc, created, lastRun)
		if err != nil {
			return err
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		// copy commands
		crows, err := src.Query("SELECT position, command FROM commands WHERE command_set_id = ? ORDER BY position ASC", id)
		if err != nil {
			return err
		}
		for crows.Next() {
			var pos int
			var cmd string
			if err := crows.Scan(&pos, &cmd); err != nil {
				crows.Close()
				return err
			}
			if _, err := dst.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", newID, pos, cmd); err != nil {
				crows.Close()
				return err
			}
		}
		crows.Close()
	}
	return nil
}
