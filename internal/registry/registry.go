package registry

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/VoxDroid/krnr/internal/nameutil"
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
// initialCommands, if provided, will be recorded as the initial version snapshot.
func (r *Repository) CreateCommandSet(name string, description *string, authorName *string, authorEmail *string, initialCommands []string) (int64, error) {
	if err := r.validateCreateName(&name); err != nil {
		return 0, err
	}
	return r.createCommandSetTx(name, description, authorName, authorEmail, initialCommands)
}

func (r *Repository) validateCreateName(name *string) error {
	*name = strings.TrimSpace(*name)
	if *name == "" {
		return fmt.Errorf("invalid name: name cannot be empty")
	}
	return nameutil.ValidateName(*name)
}

func (r *Repository) createCommandSetTx(name string, description *string, authorName *string, authorEmail *string, initialCommands []string) (int64, error) {
	// happens inside the DB engine and avoids TOCTOU races across processes.
	trx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = trx.Rollback() }()

	res, err := trx.Exec(`INSERT INTO command_sets (name, description, author_name, author_email, created_at)
			SELECT ?, ?, ?, ?, datetime('now')
			WHERE NOT EXISTS(SELECT 1 FROM command_sets WHERE TRIM(name) = ?)`, name, description, authorName, authorEmail, name)
	if err != nil {
		return 0, fmt.Errorf("insert command_set: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		// Another row with the same trimmed name already exists
		return 0, fmt.Errorf("name %q already in use", name)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// Sanity check against the row inside this transaction to ensure the inserted
	// name matches the trimmed input. If it doesn't, remove the bad row and reject.
	if err := r.validateInsertedNameTx(trx, id, name); err != nil {
		_, _ = trx.Exec("DELETE FROM command_sets WHERE id = ?", id)
		return 0, err
	}
	// insert initial commands (if any) into the commands table
	if err := r.insertInitialCommandsTx(trx, id, initialCommands); err != nil {
		return 0, err
	}
	// record an initial version (may include provided commands) inside the same transaction
	if err := r.recordVersionTx(trx, id, authorName, authorEmail, description, initialCommands, "create"); err != nil {
		return 0, err
	}
	if err := trx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) validateInsertedNameTx(trx *sql.Tx, id int64, name string) error {
	var storedName string
	row := trx.QueryRow("SELECT TRIM(name) FROM command_sets WHERE id = ?", id)
	if err := row.Scan(&storedName); err != nil {
		return fmt.Errorf("sanity check failed: %w", err)
	}
	if storedName == "" || storedName != name {
		return fmt.Errorf("sanity check failed: inserted name mismatch")
	}
	return nil
}

func (r *Repository) insertInitialCommandsTx(trx *sql.Tx, id int64, initialCommands []string) error {
	filtered := make([]string, 0, len(initialCommands))
	for _, c := range initialCommands {
		if strings.TrimSpace(c) == "" {
			continue
		}
		filtered = append(filtered, c)
	}
	for i, c := range filtered {
		if _, err := trx.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", id, i+1, c); err != nil {
			return fmt.Errorf("insert initial command: %w", err)
		}
	}
	return nil
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
	row := r.db.QueryRow("SELECT id, name, description, author_name, author_email, created_at, last_run FROM command_sets WHERE name = ?", name)
	var cs CommandSet
	if err := row.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.AuthorName, &cs.AuthorEmail, &cs.CreatedAt, &cs.LastRun); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	rows, err := r.db.Query("SELECT id, command_set_id, position, command FROM commands WHERE command_set_id = ? ORDER BY position ASC", cs.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var c Command
		if err := rows.Scan(&c.ID, &c.CommandSetID, &c.Position, &c.Command); err != nil {
			return nil, err
		}
		cs.Commands = append(cs.Commands, c)
	}

	if err := r.attachTags(&cs); err != nil {
		return nil, err
	}

	return &cs, nil
}

// ListCommandSets returns all command sets (without their commands).
func (r *Repository) ListCommandSets() ([]CommandSet, error) {
	rows, err := r.db.Query("SELECT id, name, description, author_name, author_email, created_at, last_run FROM command_sets ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []CommandSet
	for rows.Next() {
		var cs CommandSet
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.AuthorName, &cs.AuthorEmail, &cs.CreatedAt, &cs.LastRun); err != nil {
			return nil, err
		}
		if err := r.attachTags(&cs); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

// UpdateCommandSet updates a command set's metadata (name, description, author fields and tags).
// It records an update version snapshot of the current commands.
func (r *Repository) UpdateCommandSet(commandSetID int64, newName string, description *string, authorName *string, authorEmail *string, tags []string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()

	// ensure newName does not collide with another set
	if err := r.ensureNameNotTakenTx(trx, newName, commandSetID); err != nil {
		return err
	}

	// perform update
	if _, err := trx.Exec("UPDATE command_sets SET name = ?, description = ?, author_name = ?, author_email = ? WHERE id = ?", newName, description, authorName, authorEmail, commandSetID); err != nil {
		return err
	}

	// replace tags: remove existing associations and add provided ones
	if err := r.replaceTagsTx(trx, commandSetID, tags); err != nil {
		return err
	}

	// snapshot current commands for version history
	cmds, err := r.readCommandsTx(trx, commandSetID)
	if err != nil {
		return err
	}

	if err := r.recordVersionTx(trx, commandSetID, authorName, authorEmail, description, cmds, "update"); err != nil {
		return err
	}

	return trx.Commit()
}

func (r *Repository) ensureNameNotTakenTx(trx *sql.Tx, newName string, commandSetID int64) error {
	var existingID int64
	row := trx.QueryRow("SELECT id FROM command_sets WHERE name = ?", newName)
	if err := row.Scan(&existingID); err == nil {
		if existingID != commandSetID {
			return fmt.Errorf("name %q already in use", newName)
		}
	} else {
		// if Scan fails with no rows it's fine; other errors propagate
		if err != sql.ErrNoRows && err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) replaceTagsTx(trx *sql.Tx, commandSetID int64, tags []string) error {
	if _, err := trx.Exec("DELETE FROM command_set_tags WHERE command_set_id = ?", commandSetID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := trx.Exec("INSERT OR IGNORE INTO tags (name) VALUES (?)", tag); err != nil {
			return err
		}
		var tagID int64
		rrow := trx.QueryRow("SELECT id FROM tags WHERE name = ?", tag)
		if err := rrow.Scan(&tagID); err != nil {
			return err
		}
		if _, err := trx.Exec("INSERT OR IGNORE INTO command_set_tags (command_set_id, tag_id) VALUES (?, ?)", commandSetID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) readCommandsTx(trx *sql.Tx, commandSetID int64) ([]string, error) {
	rows, err := trx.Query("SELECT command FROM commands WHERE command_set_id = ? ORDER BY position ASC", commandSetID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var cmds []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	return cmds, nil
}

// UpdateCommandSetAndReplaceCommands performs an atomic metadata+commands update
// and records exactly one 'update' version representing the final state.
func (r *Repository) UpdateCommandSetAndReplaceCommands(commandSetID int64, newName string, description *string, authorName *string, authorEmail *string, tags []string, commands []string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()

	// ensure newName does not collide with another set
	if err := r.ensureNameNotTakenTx(trx, newName, commandSetID); err != nil {
		return err
	}

	// perform update and finalize transaction
	if err := r.updateMetadataAndFinalizeTx(trx, commandSetID, newName, description, authorName, authorEmail, tags, commands); err != nil {
		return err
	}
	return nil
}

func (r *Repository) replaceCommandsTx(trx *sql.Tx, commandSetID int64, commands []string) ([]string, error) {
	if _, err := trx.Exec("DELETE FROM commands WHERE command_set_id = ?", commandSetID); err != nil {
		return nil, err
	}
	pos := 1
	filtered := make([]string, 0, len(commands))
	for _, c := range commands {
		if strings.TrimSpace(c) == "" {
			continue
		}
		filtered = append(filtered, c)
		if _, err := trx.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", commandSetID, pos, c); err != nil {
			return nil, fmt.Errorf("insert command: %w", err)
		}
		pos++
	}
	return filtered, nil
}

func (r *Repository) updateMetadataAndFinalizeTx(trx *sql.Tx, commandSetID int64, newName string, description *string, authorName *string, authorEmail *string, tags []string, commands []string) error {
	if _, err := trx.Exec("UPDATE command_sets SET name = ?, description = ?, author_name = ?, author_email = ? WHERE id = ?", newName, description, authorName, authorEmail, commandSetID); err != nil {
		return err
	}
	if err := r.replaceTagsTx(trx, commandSetID, tags); err != nil {
		return err
	}
	filtered, err := r.replaceCommandsTx(trx, commandSetID, commands)
	if err != nil {
		return err
	}
	if err := r.recordVersionTx(trx, commandSetID, authorName, authorEmail, description, filtered, "update"); err != nil {
		return err
	}
	return trx.Commit()
}

// DeleteCommandSet removes a command set and its commands by name.
func (r *Repository) DeleteCommandSet(name string) error {
	trx, err := r.db.Begin()
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

	// snapshot commands before deletion
	rows, err := trx.Query("SELECT command FROM commands WHERE command_set_id = ? ORDER BY position ASC", id)
	if err != nil {
		return err
	}
	var cmds []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			_ = rows.Close()
			return err
		}
		cmds = append(cmds, c)
	}
	_ = rows.Close()
	// record deletion snapshot using the same transaction to avoid nested writes
	if err := r.recordVersionTx(trx, id, nil, nil, nil, cmds, "delete"); err != nil {
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

// ReplaceCommands replaces all commands for a given command set with the provided
// slice of command strings. Existing commands for the set are deleted and the
// new commands are inserted with positions starting at 1.
func (r *Repository) ReplaceCommands(commandSetID int64, commands []string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()

	if _, err := trx.Exec("DELETE FROM commands WHERE command_set_id = ?", commandSetID); err != nil {
		return err
	}
	for i, c := range commands {
		if _, err := trx.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", commandSetID, i+1, c); err != nil {
			return err
		}
	}
	if err := trx.Commit(); err != nil {
		return err
	}
	// record update as a new version
	_ = r.RecordVersion(commandSetID, nil, nil, nil, commands, "update")
	return nil
}

// attachTags loads tags for a command set into the provided CommandSet.
func (r *Repository) attachTags(cs *CommandSet) error {
	rows, err := r.db.Query("SELECT t.name FROM tags t JOIN command_set_tags cst ON t.id = cst.tag_id WHERE cst.command_set_id = ?", cs.ID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		cs.Tags = append(cs.Tags, name)
	}
	return nil
}

// AddTagToCommandSet adds a tag (creating it if necessary) and associates it with the command set.
func (r *Repository) AddTagToCommandSet(commandSetID int64, tag string) error {
	trx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = trx.Rollback() }()

	// ensure tag exists
	if _, err := trx.Exec("INSERT OR IGNORE INTO tags (name) VALUES (?)", tag); err != nil {
		return err
	}
	var tagID int64
	row := trx.QueryRow("SELECT id FROM tags WHERE name = ?", tag)
	if err := row.Scan(&tagID); err != nil {
		return err
	}
	// associate
	if _, err := trx.Exec("INSERT OR IGNORE INTO command_set_tags (command_set_id, tag_id) VALUES (?, ?)", commandSetID, tagID); err != nil {
		return err
	}
	return trx.Commit()
}

// RemoveTagFromCommandSet removes an association between a tag and a command set.
func (r *Repository) RemoveTagFromCommandSet(commandSetID int64, tag string) error {
	// find tag id
	row := r.db.QueryRow("SELECT id FROM tags WHERE name = ?", tag)
	var tagID int64
	if err := row.Scan(&tagID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if _, err := r.db.Exec("DELETE FROM command_set_tags WHERE command_set_id = ? AND tag_id = ?", commandSetID, tagID); err != nil {
		return err
	}
	return nil
}

// ListTagsForCommandSet returns all tag names associated with a command set.
func (r *Repository) ListTagsForCommandSet(commandSetID int64) ([]string, error) {
	rows, err := r.db.Query("SELECT t.name FROM tags t JOIN command_set_tags cst ON t.id = cst.tag_id WHERE cst.command_set_id = ?", commandSetID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, nil
}

// ListCommandSetsByTag returns all command sets that have the given tag.
func (r *Repository) ListCommandSetsByTag(tag string) ([]CommandSet, error) {
	rows, err := r.db.Query(`
		SELECT cs.id, cs.name, cs.description, cs.author_name, cs.author_email, cs.created_at, cs.last_run
		FROM command_sets cs
		JOIN command_set_tags cst ON cs.id = cst.command_set_id
		JOIN tags t ON t.id = cst.tag_id
		WHERE t.name = ?
		ORDER BY cs.created_at DESC
	`, tag)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []CommandSet
	for rows.Next() {
		var cs CommandSet
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.AuthorName, &cs.AuthorEmail, &cs.CreatedAt, &cs.LastRun); err != nil {
			return nil, err
		}
		if err := r.attachTags(&cs); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

// SearchCommandSets searches for command sets by name, description, or command content.
func (r *Repository) SearchCommandSets(query string) ([]CommandSet, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.Query(`
		SELECT DISTINCT cs.id, cs.name, cs.description, cs.author_name, cs.author_email, cs.created_at, cs.last_run
		FROM command_sets cs
		LEFT JOIN commands c ON c.command_set_id = cs.id
		WHERE cs.name LIKE ? OR cs.description LIKE ? OR c.command LIKE ?
		ORDER BY cs.created_at DESC
	`, pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []CommandSet
	for rows.Next() {
		var cs CommandSet
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.AuthorName, &cs.AuthorEmail, &cs.CreatedAt, &cs.LastRun); err != nil {
			return nil, err
		}
		if err := r.attachTags(&cs); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}
