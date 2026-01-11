package registry

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Version represents a saved snapshot of a command set.
type Version struct {
	ID           int64
	CommandSetID int64
	Version      int
	CreatedAt    string
	AuthorName   sql.NullString
	AuthorEmail  sql.NullString
	Description  sql.NullString
	Commands     []string
	Operation    string
}

// recordVersionTx writes a version record using the provided transaction. This helper
// is useful when an outer transaction is already in progress to avoid nested writes.
func (r *Repository) recordVersionTx(trx *sql.Tx, commandSetID int64, authorName *string, authorEmail *string, description *string, commands []string, operation string) error {
	cmdJSON, err := json.Marshal(commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	var maxVersion sql.NullInt64
	row := trx.QueryRow("SELECT COALESCE(MAX(version), 0) FROM command_set_versions WHERE command_set_id = ?", commandSetID)
	if err := row.Scan(&maxVersion); err != nil {
		return err
	}
	newVersion := int(maxVersion.Int64) + 1
	_, err = trx.Exec(`INSERT INTO command_set_versions
		(command_set_id, version, created_at, author_name, author_email, description, commands, operation)
		VALUES (?, ?, datetime('now'), ?, ?, ?, ?, ?)`, commandSetID, newVersion, authorName, authorEmail, description, string(cmdJSON), operation)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

// RecordVersion stores a snapshot of commands for the given command set.
func (r *Repository) RecordVersion(commandSetID int64, authorName *string, authorEmail *string, description *string, commands []string, operation string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()
	if err := r.recordVersionTx(trx, commandSetID, authorName, authorEmail, description, commands, operation); err != nil {
		return err
	}
	return trx.Commit()
}

// ListVersions returns all versions for a given command set id in descending order (newest first).
func (r *Repository) ListVersions(commandSetID int64) ([]Version, error) {
	rows, err := r.db.Query(`SELECT id, command_set_id, version, created_at, author_name, author_email, description, commands, operation
		FROM command_set_versions WHERE command_set_id = ? ORDER BY version DESC`, commandSetID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Version
	for rows.Next() {
		var v Version
		var cmdJSON string
		if err := rows.Scan(&v.ID, &v.CommandSetID, &v.Version, &v.CreatedAt, &v.AuthorName, &v.AuthorEmail, &v.Description, &cmdJSON, &v.Operation); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(cmdJSON), &v.Commands); err != nil {
			return nil, fmt.Errorf("unmarshal commands: %w", err)
		}
		out = append(out, v)
	}
	return out, nil
}

// ListVersionsByName finds the command set by name and returns its versions.
func (r *Repository) ListVersionsByName(name string) ([]Version, error) {
	row := r.db.QueryRow("SELECT id FROM command_sets WHERE name = ?", name)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.ListVersions(id)
}

// GetVersion returns a specific version entry for a command set id and version number.
func (r *Repository) GetVersion(commandSetID int64, versionNum int) (*Version, error) {
	row := r.db.QueryRow(`SELECT id, command_set_id, version, created_at, author_name, author_email, description, commands, operation
		FROM command_set_versions WHERE command_set_id = ? AND version = ?`, commandSetID, versionNum)
	var v Version
	var cmdJSON string
	if err := row.Scan(&v.ID, &v.CommandSetID, &v.Version, &v.CreatedAt, &v.AuthorName, &v.AuthorEmail, &v.Description, &cmdJSON, &v.Operation); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(cmdJSON), &v.Commands); err != nil {
		return nil, fmt.Errorf("unmarshal commands: %w", err)
	}
	return &v, nil
}

// ApplyVersionByName replaces the current commands for the named command set with the
// specified version's commands and records a new 'rollback' version.
func (r *Repository) ApplyVersionByName(name string, versionNum int) error {
	cs, err := r.GetCommandSetByName(name)
	if err != nil {
		return err
	}
	if cs == nil {
		return fmt.Errorf("command set not found: %s", name)
	}
	v, err := r.GetVersion(cs.ID, versionNum)
	if err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("version %d not found for %s", versionNum, name)
	}
	// replace commands
	if err := r.ReplaceCommands(cs.ID, v.Commands); err != nil {
		return err
	}
	// record rollback as a new version
	return r.RecordVersion(cs.ID, nil, nil, nil, v.Commands, "rollback")
}
