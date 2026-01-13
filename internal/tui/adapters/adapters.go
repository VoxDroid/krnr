// Package adapters provides adapter interfaces and lightweight types used by
// the TUI to decouple it from the internal domain packages.
package adapters

import (
	"context"
	"errors"
)

// ErrNotFound is used when a requested item cannot be found in the repository.
var ErrNotFound = errors.New("not found")

// CommandSetSummary represents a lightweight summary of a command set used by the TUI.
type CommandSetSummary struct {
	Name        string
	Description string
	Version     string
	Commands    []string
	AuthorName  string
	AuthorEmail string
	Tags        []string
	CreatedAt   string
	LastRun     string
}

// Version mirrors registry.Version and is used by the TUI to render history entries.
type Version struct {
	Version     int
	CreatedAt   string
	AuthorName  string
	AuthorEmail string
	Description string
	Commands    []string
	Operation   string
}

// RunEvent represents streaming output from a running commandset.
type RunEvent struct {
	Line string
	Err  error
}

// RunHandle is returned by ExecutorAdapter.Run to manage streaming output and cancellation.
type RunHandle interface {
	// Events returns a receive-only channel for streaming output.
	Events() <-chan RunEvent
	// Cancel requests termination of the running command.
	Cancel()
}

// RegistryAdapter describes the minimal subset of registry operations used by the UI.
// Keep methods small and easy to mock for tests.
type RegistryAdapter interface {
	ListCommandSets(ctx context.Context) ([]CommandSetSummary, error)
	GetCommandSet(ctx context.Context, name string) (CommandSetSummary, error)
	GetCommands(ctx context.Context, name string) ([]string, error)
	SaveCommandSet(ctx context.Context, cs CommandSetSummary) error
	DeleteCommandSet(ctx context.Context, name string) error
	// ReplaceCommands replaces the commands for an existing command set.
	ReplaceCommands(ctx context.Context, name string, commands []string) error
	// UpdateCommandSet updates metadata and tags for an existing command set.
	UpdateCommandSet(ctx context.Context, oldName string, cs CommandSetSummary) error
	// ListVersionsByName returns the versions for a command set (newest first)
	ListVersionsByName(ctx context.Context, name string) ([]Version, error)
	// ApplyVersionByName applies the specified historic version to the named command set (rollback)
	ApplyVersionByName(ctx context.Context, name string, versionNum int) error
}

// ExecutorAdapter describes running and streaming commandset executions.
// The `commands` slice is the list of shell commands to execute sequentially.
type ExecutorAdapter interface {
	Run(ctx context.Context, name string, commands []string) (RunHandle, error)
}

// ImportExportAdapter describes import/export operations.
type ImportExportAdapter interface {
	Export(ctx context.Context, name string, dest string) error
	Import(ctx context.Context, src string, policy string) error
}

// InstallerAdapter minimal interface for install/uninstall
type InstallerAdapter interface {
	Install(ctx context.Context, name string) error
	Uninstall(ctx context.Context, name string) error
}
