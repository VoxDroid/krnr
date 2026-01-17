package model

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"database/sql"

	"github.com/VoxDroid/krnr/internal/config"
	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/importer"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

func createAndPopulateSrcDB(t *testing.T, src string) {
	srcConn2, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatalf("open src2: %v", err)
	}
	defer func() { _ = srcConn2.Close() }()
	if err := db.ApplyMigrations(srcConn2); err != nil {
		t.Fatalf("apply migrations src: %v", err)
	}
	srcRepo := registry.NewRepository(srcConn2)
	if _, err := srcRepo.CreateCommandSet("imported-db", nil, nil, nil, []string{"echo hi"}); err != nil {
		t.Fatalf("create command set src: %v", err)
	}
}

func TestImportDatabaseReopen(t *testing.T) {
	// set a temp DB path
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "krnr.db")
	src := filepath.Join(tmp, "src.db")
	_ = os.Setenv(config.EnvKRNRDB, dst)
	defer func() { _ = os.Unsetenv(config.EnvKRNRDB) }()

	// create a source DB with one command set
	srcDB, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatalf("open src db: %v", err)
	}
	_ = srcDB.Close()
	// apply migrations and write a command set via a temporary repo
	srcConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("init src db: %v", err)
	}
	_ = srcConn.Close()

	createAndPopulateSrcDB(t, src)

	// ensure initial active DB exists (empty)
	activeConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("init active db: %v", err)
	}
	defer func() { _ = activeConn.Close() }()

	// New model backed by active DB
	r := registry.NewRepository(activeConn)
	m := New(adapters.NewRegistryAdapter(r), nil, adapters.NewImportExportAdapter(activeConn), nil)

	// Import source into active DB (overwrite)
	if err := importer.ImportDatabase(src, true, importer.ImportOptions{}); err != nil {
		t.Fatalf("import database failed: %v", err)
	}

	// After import, reopen DB and refresh list
	if err := m.ReopenDB(context.Background()); err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if err := m.RefreshList(context.Background()); err != nil {
		t.Fatalf("refresh list: %v", err)
	}
	defer func() { _ = m.Close() }()

	found := false
	for _, s := range m.ListCached() {
		if s.Name == "imported-db" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected imported-db to be present in list, got %v", m.ListCached())
	}
}
