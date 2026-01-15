package ui

import (
	"context"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// UIModel defines the small subset of methods from the framework-agnostic
// internal UI model that the TUI depends on. This decouples presentation
// code from the concrete implementation and makes unit testing easier.
type UIModel interface {
	RefreshList(ctx context.Context) error
	ListCached() []adapters.CommandSetSummary
	GetCommandSet(ctx context.Context, name string) (adapters.CommandSetSummary, error)
	ListVersions(ctx context.Context, name string) ([]adapters.Version, error)
	ApplyVersion(ctx context.Context, name string, version int) error
	Delete(ctx context.Context, name string) error
	Export(ctx context.Context, name string, dest string) error
	UpdateCommandSet(ctx context.Context, oldName string, cs adapters.CommandSetSummary) error
	ReplaceCommands(ctx context.Context, name string, cmds []string) error
	Run(ctx context.Context, name string, _ []string) (adapters.RunHandle, error)
	Save(ctx context.Context, cs adapters.CommandSetSummary) error
	Install(ctx context.Context, name string) error
	Uninstall(ctx context.Context, name string) error
}
