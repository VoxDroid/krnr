// Package model provides a framework-agnostic UI model built on top of
// adapter interfaces so the TUI code can remain presentation-focused.
package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/install"
	"github.com/VoxDroid/krnr/internal/nameutil"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// ErrNotFound is returned when a requested command set cannot be found.
var ErrNotFound = errors.New("not found")

// UIModel is a framework-agnostic model for screens and actions.
// It depends only on adapter interfaces.
type UIModel struct {
	registry  adapters.RegistryAdapter
	executor  adapters.ExecutorAdapter
	impExp    adapters.ImportExportAdapter
	installer adapters.InstallerAdapter

	cache []adapters.CommandSetSummary
	// serialize Save/Update operations to avoid races when multiple
	// concurrent saves are attempted (defensive, DB-enforced as well)
	saveMu sync.Mutex
}

// New constructs a UIModel backed by the provided adapters.
func New(reg adapters.RegistryAdapter, ex adapters.ExecutorAdapter, ie adapters.ImportExportAdapter, inst adapters.InstallerAdapter) *UIModel {
	return &UIModel{registry: reg, executor: ex, impExp: ie, installer: inst}
}

// RefreshList fetches the commandset list and caches it.
func (m *UIModel) RefreshList(ctx context.Context) error {
	list, err := m.registry.ListCommandSets(ctx)
	if err != nil {
		return err
	}
	m.cache = list
	return nil
}

// ListCached returns the cached command set summaries.
func (m *UIModel) ListCached() []adapters.CommandSetSummary { return m.cache }

// FindByName searches the cache for a command set by name.
func (m *UIModel) FindByName(name string) (adapters.CommandSetSummary, error) {
	for _, cs := range m.cache {
		if cs.Name == name {
			return cs, nil
		}
	}
	return adapters.CommandSetSummary{}, ErrNotFound
}

// GetCommandSet fetches the full command set metadata, including commands,
// from the underlying registry adapter. This is used by the UI to display
// full previews when a set is selected.
func (m *UIModel) GetCommandSet(ctx context.Context, name string) (adapters.CommandSetSummary, error) {
	return m.registry.GetCommandSet(ctx, name)
}

// Run starts execution and returns a handle for streaming events.
// Run starts execution and returns a handle for streaming events.
func (m *UIModel) Run(ctx context.Context, name string, _ []string) (adapters.RunHandle, error) {
	// fetch commands for the set
	cmds, err := m.registry.GetCommands(ctx, name)
	if err != nil {
		return nil, err
	}
	// params/args handling is TODO — for now, ignore args and run the commands
	return m.executor.Run(ctx, name, cmds)
}

// ReplaceCommands replaces the commands for an existing command set by name.
func (m *UIModel) ReplaceCommands(ctx context.Context, name string, commands []string) error {
	return m.registry.ReplaceCommands(ctx, name, commands)
}

// UpdateCommandSet updates metadata (name, description, author, tags) for an existing set
func (m *UIModel) UpdateCommandSet(ctx context.Context, oldName string, cs adapters.CommandSetSummary) error {
	// serialize updates that can change names to avoid races with concurrent saves
	m.saveMu.Lock()
	defer m.saveMu.Unlock()
	return m.registry.UpdateCommandSet(ctx, oldName, cs)
}

// UpdateCommandSetAndReplaceCommands performs an atomic metadata+commands update
// and records a single 'update' version representing the final state.
func (m *UIModel) UpdateCommandSetAndReplaceCommands(ctx context.Context, oldName string, cs adapters.CommandSetSummary) error {
	// serialize updates that can change names to avoid races with concurrent saves
	m.saveMu.Lock()
	defer m.saveMu.Unlock()
	return m.registry.UpdateCommandSetAndReplaceCommands(ctx, oldName, cs)
}

// ListVersions fetches historical versions for a command set by name (newest first)
func (m *UIModel) ListVersions(ctx context.Context, name string) ([]adapters.Version, error) {
	return m.registry.ListVersionsByName(ctx, name)
}

// ApplyVersion applies a historic version to the current command set (rollback)
func (m *UIModel) ApplyVersion(ctx context.Context, name string, versionNum int) error {
	return m.registry.ApplyVersionByName(ctx, name, versionNum)
}

// DeleteVersion deletes a specific version record for the named command set
func (m *UIModel) DeleteVersion(ctx context.Context, name string, versionNum int) error {
	return m.registry.DeleteVersionByName(ctx, name, versionNum)
}

// Delete removes a named command set from the registry
func (m *UIModel) Delete(ctx context.Context, name string) error {
	return m.registry.DeleteCommandSet(ctx, name)
}

// Export an existing commandset to dest path
func (m *UIModel) Export(ctx context.Context, name string, dest string) error {
	if m.impExp == nil {
		return fmt.Errorf("import/export adapter not configured")
	}
	// When name is empty, we are exporting the whole database, so skip
	// validating a command set exists. For named exports, ensure the set
	// exists before delegating to the adapter.
	if name != "" {
		if _, err := m.registry.GetCommandSet(ctx, name); err != nil {
			return err
		}
	}
	return m.impExp.Export(ctx, name, dest)
}

// ImportSet imports a command-set file using the given conflict policy and
// dedupe option.
func (m *UIModel) ImportSet(ctx context.Context, src string, policy string, dedupe bool) error {
	if m.impExp == nil {
		return fmt.Errorf("import/export adapter not configured")
	}
	return m.impExp.ImportSet(ctx, src, policy, dedupe)
}

// ImportDB imports a database file into the active DB. If overwrite is true
// it replaces the active DB file.
func (m *UIModel) ImportDB(ctx context.Context, src string, overwrite bool) error {
	if m.impExp == nil {
		return fmt.Errorf("import/export adapter not configured")
	}
	return m.impExp.ImportDB(ctx, src, overwrite)
}

// ReopenDB re-initializes the registry adapter using a fresh DB connection.
// This is useful after a full DB overwrite which replaces the on-disk file;
// existing SQL connections may continue to reference the old file contents.
func (m *UIModel) ReopenDB(ctx context.Context) error {
	if m.registry == nil {
		return fmt.Errorf("registry adapter not configured")
	}
	// If the current registry adapter is not the concrete implementation
	// used in production (i.e., a test fake), avoid re-opening a real DB
	// connection so tests that inject fake registries continue to work.
	if _, ok := m.registry.(*adapters.RegistryAdapterImpl); !ok {
		// just refresh the list from whatever adapter is present
		return m.RefreshList(ctx)
	}
	// attempt to close the existing adapter's DB connection if it exposes Close
	if c, ok := m.registry.(interface{ Close() error }); ok {
		_ = c.Close()
	}
	// Obtain a fresh DB connection and create a new repository/adapter.
	dbConn, err := db.InitDB()
	if err != nil {
		return err
	}
	repo := registry.NewRepository(dbConn)
	m.registry = adapters.NewRegistryAdapter(repo)
	// Refresh cache with new adapter
	return m.RefreshList(ctx)
}

// Close cleans up any resources held by the UIModel (e.g., DB connections).
func (m *UIModel) Close() error {
	if m.registry == nil {
		return nil
	}
	if c, ok := m.registry.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

// Install / Uninstall
// Install performs an installation of the krnr binary using the provided options.
func (m *UIModel) Install(ctx context.Context, opts install.Options) ([]string, error) {
	if m.installer == nil {
		return nil, fmt.Errorf("installer adapter not configured")
	}
	return m.installer.Install(ctx, opts)
}

// Uninstall removes an installed krnr from the host and returns actions performed.
func (m *UIModel) Uninstall(ctx context.Context) ([]string, error) {
	if m.installer == nil {
		return nil, fmt.Errorf("installer adapter not configured")
	}
	return m.installer.Uninstall(ctx)
}

// Save creates a new command set from provided metadata and commands
func (m *UIModel) Save(ctx context.Context, cs adapters.CommandSetSummary) error {
	m.saveMu.Lock()
	defer m.saveMu.Unlock()

	// sanitize names coming from adapters/UI
	if s, changed := nameutil.SanitizeName(cs.Name); changed {
		cs.Name = s
	}
	name := strings.TrimSpace(cs.Name)
	if name == "" {
		return fmt.Errorf("invalid name: name cannot be empty")
	}
	// Validate characters (reject control bytes/non-UTF8)
	if err := nameutil.ValidateName(name); err != nil {
		return err
	}
	cs.Name = name
	// Do not allow creating duplicate names; check under lock to avoid TOCTOU races
	if _, err := m.registry.GetCommandSet(ctx, cs.Name); err == nil {
		return fmt.Errorf("invalid name: name already exists")
	} else if err != nil && err != adapters.ErrNotFound {
		return err
	}
	return m.registry.SaveCommandSet(ctx, cs)
}

// FakeRunHandle simulates a streaming RunHandle for tests.
func FakeRunHandle(lines []string, delay time.Duration) adapters.RunHandle {
	events := make(chan adapters.RunEvent)
	go func() {
		defer close(events)
		for _, l := range lines {
			events <- adapters.RunEvent{Line: l}
			if delay > 0 {
				time.Sleep(delay)
			}
		}
	}()
	return &fakeRunHandle{ch: events}
}

type fakeRunHandle struct{ ch <-chan adapters.RunEvent }

func (f *fakeRunHandle) Events() <-chan adapters.RunEvent { return f.ch }
func (f *fakeRunHandle) Cancel()                          {}
func (f *fakeRunHandle) WriteInput(p []byte) (int, error) { return len(p), nil }
