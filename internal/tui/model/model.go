package model

import (
	"context"
	"errors"
	"time"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

var ErrNotFound = errors.New("not found")

// UIModel is a framework-agnostic model for screens and actions.
// It depends only on adapter interfaces.
type UIModel struct {
	registry adapters.RegistryAdapter
	executor adapters.ExecutorAdapter
	impExp   adapters.ImportExportAdapter
	installer adapters.InstallerAdapter

	cache []adapters.CommandSetSummary
}

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

func (m *UIModel) ListCached() []adapters.CommandSetSummary { return m.cache }

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
func (m *UIModel) Run(ctx context.Context, name string, args []string) (adapters.RunHandle, error) {
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

// Export an existing commandset to dest path
func (m *UIModel) Export(ctx context.Context, name string, dest string) error {
	_, err := m.registry.GetCommandSet(ctx, name)
	if err != nil {
		return err
	}
	return m.impExp.Export(ctx, name, dest)
}

// Import imports a file and returns after completion (blocking)
func (m *UIModel) Import(ctx context.Context, src string, policy string) error {
	return m.impExp.Import(ctx, src, policy)
}

// Install / Uninstall
func (m *UIModel) Install(ctx context.Context, name string) error { return m.installer.Install(ctx, name) }
func (m *UIModel) Uninstall(ctx context.Context, name string) error { return m.installer.Uninstall(ctx, name) }

// Save creates a new command set from provided metadata and commands
func (m *UIModel) Save(ctx context.Context, cs adapters.CommandSetSummary) error {
	return m.registry.SaveCommandSet(ctx, cs)
}

// Simple helper to simulate streaming in tests
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
func (f *fakeRunHandle) Cancel() {}
