package adapters

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

// Minimal types mirroring internal domain objects. We keep these small to avoid cycles.
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
