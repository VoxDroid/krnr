package ui

import (
	"context"

	"github.com/VoxDroid/krnr/internal/install"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// Model defines the small subset of methods from the framework-agnostic
// internal UI model that the TUI depends on. This decouples presentation
// code from the concrete implementation and makes unit testing easier.
//
// Named `Model` (instead of `UIModel`) to avoid redundant package/type
// stuttering when referenced as `ui.Model`.
type Model interface {
	RefreshList(ctx context.Context) error
	ListCached() []adapters.CommandSetSummary
	GetCommandSet(ctx context.Context, name string) (adapters.CommandSetSummary, error)
	ListVersions(ctx context.Context, name string) ([]adapters.Version, error)
	ApplyVersion(ctx context.Context, name string, version int) error
	Delete(ctx context.Context, name string) error
	Export(ctx context.Context, name string, dest string) error
	ImportSet(ctx context.Context, src string, policy string, dedupe bool) error
	ImportDB(ctx context.Context, src string, overwrite bool) error
	// ReopenDB forces the model to re-open the underlying DB connection and
	// refresh its registry adapter. This is necessary after a full DB file
	// overwrite so subsequent queries read the new file contents.
	ReopenDB(ctx context.Context) error
	// Close cleans up any resources held by the model (e.g., DB connections).
	Close() error
	UpdateCommandSet(ctx context.Context, oldName string, cs adapters.CommandSetSummary) error
	ReplaceCommands(ctx context.Context, name string, cmds []string) error
	Run(ctx context.Context, name string, _ []string) (adapters.RunHandle, error)
	Save(ctx context.Context, cs adapters.CommandSetSummary) error
	Install(ctx context.Context, opts install.Options) ([]string, error)
	Uninstall(ctx context.Context) ([]string, error)
}
