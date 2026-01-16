package model

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/install"
	"github.com/VoxDroid/krnr/internal/registry"
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

func TestSaveRejectsEmptyName(t *testing.T) {
	// registry that records saves
	rec := &saveRecorder{}
	m := New(rec, &testExecutor{}, &testImportExport{}, &testInstaller{})
	// try to save with empty name
	err := m.Save(context.Background(), adapters.CommandSetSummary{Name: "   "})
	if err == nil || !strings.Contains(err.Error(), "invalid name") {
		t.Fatalf("expected invalid name error, got %v", err)
	}
	if rec.last.Name != "" {
		t.Fatalf("expected registry Save not to be called, got %#v", rec.last)
	}
}

func TestSaveRejectsDuplicateName(t *testing.T) {
	// test registry pre-populated with an existing name
	fake := &testRegistry{items: []adapters.CommandSetSummary{{Name: "exists"}}}
	m := New(fake, &testExecutor{}, &testImportExport{}, &testInstaller{})
	err := m.Save(context.Background(), adapters.CommandSetSummary{Name: "exists"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate-name error, got %v", err)
	}
}

func TestConcurrentSavesDoNotCreateDuplicates(t *testing.T) {
	// use a real repository-backed adapter to simulate real DB behavior
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	regAdapter := adapters.NewRegistryAdapter(r)
	m := New(regAdapter, &testExecutor{}, &testImportExport{}, &testInstaller{})

	// clean
	_ = r.DeleteCommandSet("concur")

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = m.Save(context.Background(), adapters.CommandSetSummary{Name: "concur"})
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, e := range errs {
		if e == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful save, got %d (errs=%#v)", successes, errs)
	}
	// cleanup
	_ = r.DeleteCommandSet("concur")
}

type saveRecorder struct{ last adapters.CommandSetSummary }

func (s *saveRecorder) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return nil, nil
}
func (s *saveRecorder) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return adapters.CommandSetSummary{}, adapters.ErrNotFound
}
func (s *saveRecorder) SaveCommandSet(_ context.Context, cs adapters.CommandSetSummary) error {
	s.last = cs
	return nil
}
func (s *saveRecorder) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (s *saveRecorder) GetCommands(_ context.Context, _ string) ([]string, error) {
	return nil, adapters.ErrNotFound
}
func (s *saveRecorder) ReplaceCommands(_ context.Context, _ string, _ []string) error { return nil }
func (s *saveRecorder) UpdateCommandSet(_ context.Context, _ string, _ adapters.CommandSetSummary) error {
	return nil
}
func (s *saveRecorder) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (s *saveRecorder) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }

// test helpers
type testRegistry struct{ items []adapters.CommandSetSummary }

func (t *testRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return t.items, nil
}

func (t *testRegistry) ReopenDB(ctx context.Context) error {
	// noop for test registry as it does not maintain a DB connection
	return nil
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

func (t *testImportExport) Export(_ context.Context, _ string, _ string) error            { return nil }
func (t *testImportExport) ImportSet(_ context.Context, _ string, _ string, _ bool) error { return nil }
func (t *testImportExport) ImportDB(_ context.Context, _ string, _ bool) error            { return nil }

type testInstaller struct{}

func (t *testInstaller) Install(_ context.Context, _ install.Options) ([]string, error) { return []string{"installed (test)"}, nil }
func (t *testInstaller) Uninstall(_ context.Context) ([]string, error) { return []string{"uninstalled (test)"}, nil }
