package model

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

func TestRefreshListAndFind(t *testing.T) {
	reg := &adapters.CommandSetSummary{}
	_ = reg
	// use fake registry
	fake := &testRegistry{items: []adapters.CommandSetSummary{{Name: "one"}, {Name: "two"}}}
	m := New(fake, &testExecutor{}, &testImportExport{}, &testInstaller{})
	if err := m.RefreshList(context.Background()); err != nil { t.Fatalf("refresh failed: %v", err) }
	if len(m.ListCached()) != 2 { t.Fatalf("expected 2 cached items") }
	if _, err := m.FindByName("one"); err != nil { t.Fatalf("expected to find 'one' %v", err) }
}

// test helpers
type testRegistry struct{ items []adapters.CommandSetSummary }
func (t *testRegistry) ListCommandSets(ctx context.Context) ([]adapters.CommandSetSummary, error) { return t.items, nil }
func (t *testRegistry) GetCommandSet(ctx context.Context, name string) (adapters.CommandSetSummary, error) {
	for _, it := range t.items { if it.Name == name { return it, nil }}
	return adapters.CommandSetSummary{}, ErrNotFound
}
func (t *testRegistry) SaveCommandSet(ctx context.Context, cs adapters.CommandSetSummary) error { return nil }
func (t *testRegistry) DeleteCommandSet(ctx context.Context, name string) error { return nil }
func (t *testRegistry) GetCommands(ctx context.Context, name string) ([]string, error) {
	// return a trivial command list for tests
	return []string{"echo ok"}, nil
}
func (t *testRegistry) ReplaceCommands(ctx context.Context, name string, commands []string) error { return nil }

type testExecutor struct{}
func (t *testExecutor) Run(ctx context.Context, name string, commands []string) (adapters.RunHandle, error) { return FakeRunHandle([]string{"ok"}, 0), nil }

type testImportExport struct{}
func (t *testImportExport) Export(ctx context.Context, name string, dest string) error { return nil }
func (t *testImportExport) Import(ctx context.Context, src string, policy string) error { return nil }

type testInstaller struct{}
func (t *testInstaller) Install(ctx context.Context, name string) error { return nil }
func (t *testInstaller) Uninstall(ctx context.Context, name string) error { return nil }
