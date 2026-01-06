package registry

import (
	"database/sql"
	"fmt"
)

// Repository provides CRUD operations for command sets and commands.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Repository using db.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateCommandSet inserts a new command set and returns its ID.
func (r *Repository) CreateCommandSet(name string, description *string) (int64, error) {
	rx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer trx.Rollback()

	res, err := trx.Exec("INSERT INTO command_sets (name, description, created_at) VALUES (?, ?, datetime('now'))", name, description)
	if err != nil {
		return 0, fmt.Errorf("insert command_set: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := trx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// AddCommand adds a command to a command set at the given position.
func (r *Repository) AddCommand(commandSetID int64, position int, cmd string) (int64, error) {
	res, err := r.db.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", commandSetID, position, cmd)
	if err != nil {
		return 0, fmt.Errorf("insert command: %w", err)
	}
	return res.LastInsertId()
}

// GetCommandSetByName retrieves a command set and its commands by name.
func (r *Repository) GetCommandSetByName(name string) (*CommandSet, error) {
	row := r.db.QueryRow("SELECT id, name, description, created_at, last_run FROM command_sets WHERE name = ?", name)
	var cs CommandSet
	if err := row.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.CreatedAt, &cs.LastRun); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	rows, err := r.db.Query("SELECT id, command_set_id, position, command FROM commands WHERE command_set_id = ? ORDER BY position ASC", cs.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c Command
		if err := rows.Scan(&c.ID, &c.CommandSetID, &c.Position, &c.Command); err != nil {
			return nil, err
		}
		cs.Commands = append(cs.Commands, c)
	}
	return &cs, nil
}

// ListCommandSets returns all command sets (without their commands).
func (r *Repository) ListCommandSets() ([]CommandSet, error) {
	rows, err := r.db.Query("SELECT id, name, description, created_at, last_run FROM command_sets ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CommandSet
	for rows.Next() {
		var cs CommandSet
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.CreatedAt, &cs.LastRun); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

// DeleteCommandSet removes a command set and its commands by name.
func (r *Repository) DeleteCommandSet(name string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer trx.Rollback()

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
