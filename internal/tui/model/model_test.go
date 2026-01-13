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
	if err := m.RefreshList(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if len(m.ListCached()) != 2 {
		t.Fatalf("expected 2 cached items")
	}
	if _, err := m.FindByName("one"); err != nil {
		t.Fatalf("expected to find 'one' %v", err)
	}
}

// test helpers
type testRegistry struct{ items []adapters.CommandSetSummary }

func (t *testRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return t.items, nil
}
func (t *testRegistry) GetCommandSet(_ context.Context, name string) (adapters.CommandSetSummary, error) {
	for _, it := range t.items {
		if it.Name == name {
			return it, nil
		}
	}
	return adapters.CommandSetSummary{}, ErrNotFound
}
func (t *testRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error {
	return nil
}
func (t *testRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (t *testRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	// return a trivial command list for tests
	return []string{"echo ok"}, nil
}
func (t *testRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error {
	return nil
}
func (t *testRegistry) UpdateCommandSet(_ context.Context, _ string, _ adapters.CommandSetSummary) error {
	return nil
}
func (t *testRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (t *testRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error {
	return nil
}

type testExecutor struct{}

func (t *testExecutor) Run(_ context.Context, _ string, _ []string) (adapters.RunHandle, error) {
	return FakeRunHandle([]string{"ok"}, 0), nil
}

type testImportExport struct{}

func (t *testImportExport) Export(_ context.Context, _ string, _ string) error  { return nil }
func (t *testImportExport) Import(_ context.Context, _ string, _ string) error { return nil }

type testInstaller struct{}

func (t *testInstaller) Install(_ context.Context, _ string) error   { return nil }
func (t *testInstaller) Uninstall(_ context.Context, _ string) error { return nil }
