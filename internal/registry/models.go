// Package registry provides command registry functionality.
package registry

import "database/sql"

// CommandSet represents a named workflow.
type CommandSet struct {
	ID          int64
	Name        string
	Description sql.NullString
	AuthorName  sql.NullString
	AuthorEmail sql.NullString
	CreatedAt   string
	LastRun     sql.NullString
	Commands    []Command
	Tags        []string
}

// Command is a single shell command within a CommandSet.
type Command struct {
	ID           int64
	CommandSetID int64
	Position     int
	Command      string
}
