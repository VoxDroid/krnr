package importer

import "database/sql"

// copyCommands copies commands for a given source command set id into dst using new command set id
func copyCommands(src *sql.DB, srcCommandSetID int64, dst *sql.DB, newID int64) error {
	crows, err := src.Query("SELECT position, command FROM commands WHERE command_set_id = ? ORDER BY position ASC", srcCommandSetID)
	if err != nil {
		return err
	}
	defer func() { _ = crows.Close() }()
	for crows.Next() {
		var pos int
		var cmd string
		if err := crows.Scan(&pos, &cmd); err != nil {
			return err
		}
		if _, err := dst.Exec("INSERT INTO commands (command_set_id, position, command) VALUES (?, ?, ?)", newID, pos, cmd); err != nil {
			return err
		}
	}
	return nil
}
