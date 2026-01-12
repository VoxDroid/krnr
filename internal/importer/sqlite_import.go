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
	"github.com/VoxDroid/krnr/internal/registry"
)

// ImportOptions controls per-set conflict behavior during import.
type ImportOptions struct {
	OnConflict string // rename|skip|overwrite|merge
	Dedupe     bool   // dedupe identical commands when merging
}

// ImportDatabase copies srcPath into the default database location when overwrite
// is true. If overwrite is false and opts.OnConflict == "rename" (default),
// the function returns an error. If overwrite is false and a per-set policy is
// provided in opts, the function will merge the command sets from src into the
// existing DB applying the per-set policy.
func ImportDatabase(srcPath string, overwrite bool, opts ImportOptions) error {
	dst, err := config.DBPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil && !overwrite {
		// If the caller explicitly set a per-set policy, allow merging/importing
		// into the existing DB; otherwise, reject to avoid accidental overwrite.
		if opts.OnConflict == "" || opts.OnConflict == "rename" {
			return errors.New("destination database exists; use overwrite=true to replace or specify --on-conflict to merge")
		}
		// proceed to per-set import from src into dst
		return ImportCommandSet(srcPath, opts)
	}
	// overwrite path: copy file
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

// ImportCommandSet imports all command sets from srcPath into the active DB.
// Options control how name conflicts are handled.
func ImportCommandSet(srcPath string, opts ImportOptions) error {
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

	r := registry.NewRepository(dst)

	// map policies to handler closures to reduce complexity
	handlers := map[string]func(int64, string, sql.NullString, string, sql.NullString) error{
		"rename": func(id int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
			return importWithRename(dst, src, id, name, desc, created, lastRun)
		},
		"skip": func(id int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
			return importWithSkip(dst, src, id, name, desc, created, lastRun)
		},
		"overwrite": func(id int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
			return importWithOverwrite(dst, src, id, name, desc, created, lastRun)
		},
		"merge": func(id int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
			return importWithMerge(r, dst, src, id, name, desc, created, lastRun, opts)
		},
	}

	policy := opts.OnConflict
	if policy == "" {
		policy = "rename"
	}

	for rows.Next() {
		var id int64
		var name string
		var desc sql.NullString
		var created string
		var lastRun sql.NullString
		if err := rows.Scan(&id, &name, &desc, &created, &lastRun); err != nil {
			return err
		}
		h, ok := handlers[policy]
		if !ok {
			return fmt.Errorf("unknown on-conflict policy: %s", policy)
		}
		if err := h(id, name, desc, created, lastRun); err != nil {
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

func commandSetExists(dst *sql.DB, name string) (bool, error) {
	var cnt int
	r := dst.QueryRow("SELECT count(*) FROM command_sets WHERE name = ?", name)
	if err := r.Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func deleteCommandSetByName(dst *sql.DB, name string) error {
	trx, err := dst.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()
	var id int64
	row := trx.QueryRow("SELECT id FROM command_sets WHERE name = ?", name)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if _, err := trx.Exec("DELETE FROM commands WHERE command_set_id = ?", id); err != nil {
		return err
	}
	if _, err := trx.Exec("DELETE FROM command_sets WHERE id = ?", id); err != nil {
		return err
	}
	return trx.Commit()
}

func getCommands(src *sql.DB, srcID int64) ([]string, error) {
	rows, err := src.Query("SELECT command FROM commands WHERE command_set_id = ? ORDER BY position ASC", srcID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// helper imports
func importWithRename(dst *sql.DB, src *sql.DB, srcID int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
	uName, err := ensureUniqueName(dst, name)
	if err != nil {
		return err
	}
	newID, err := insertCommandSet(dst, uName, desc, created, lastRun)
	if err != nil {
		return err
	}
	return copyCommands(src, srcID, dst, newID)
}

func importWithSkip(dst *sql.DB, src *sql.DB, srcID int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
	exists, err := commandSetExists(dst, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	newID, err := insertCommandSet(dst, name, desc, created, lastRun)
	if err != nil {
		return err
	}
	return copyCommands(src, srcID, dst, newID)
}

func importWithOverwrite(dst *sql.DB, src *sql.DB, srcID int64, name string, desc sql.NullString, created string, lastRun sql.NullString) error {
	if err := deleteCommandSetByName(dst, name); err != nil {
		return err
	}
	newID, err := insertCommandSet(dst, name, desc, created, lastRun)
	if err != nil {
		return err
	}
	return copyCommands(src, srcID, dst, newID)
}

func importWithMerge(r *registry.Repository, dst *sql.DB, src *sql.DB, srcID int64, name string, desc sql.NullString, created string, lastRun sql.NullString, opts ImportOptions) error {
	// if set doesn't exist: insert directly
	existing, err := r.GetCommandSetByName(name)
	if err != nil {
		return err
	}
	if existing == nil {
		newID, err := insertCommandSet(dst, name, desc, created, lastRun)
		if err != nil {
			return err
		}
		return copyCommands(src, srcID, dst, newID)
	}

	inc, err := getCommands(src, srcID)
	if err != nil {
		return err
	}
	merged := mergeCommands(existing.Commands, inc, opts.Dedupe)
	return r.ReplaceCommands(existing.ID, merged)
}

func mergeCommands(existing []registry.Command, incoming []string, dedupe bool) []string {
	out := make([]string, 0, len(existing)+len(incoming))
	for _, c := range existing {
		out = append(out, c.Command)
	}
	if dedupe {
		seen := map[string]bool{}
		for _, c := range out {
			seen[c] = true
		}
		for _, c := range incoming {
			if !seen[c] {
				out = append(out, c)
				seen[c] = true
			}
		}
	} else {
		out = append(out, incoming...)
	}
	return out
}
