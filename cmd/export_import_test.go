package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/exporter"
	"github.com/VoxDroid/krnr/internal/importer"
	"github.com/VoxDroid/krnr/internal/registry"
)

// test helpers
func exportSetFile(t *testing.T, setName string, cmds []string) string {
	t.Helper()
	tmp := setupTempDB(t)
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	r := registry.NewRepository(dbConn)
	desc := setName
	id, err := r.CreateCommandSet(setName, &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	for i, c := range cmds {
		if _, err := r.AddCommand(id, i+1, c); err != nil {
			t.Fatalf("AddCommand: %v", err)
		}
	}
	dst := filepath.Join(tmp, setName+".db")
	if err := exporter.ExportCommandSet(dbConn, setName, dst); err != nil {
		t.Fatalf("ExportCommandSet: %v", err)
	}
	_ = dbConn.Close()
	return dst
}

func importIntoFreshHome(t *testing.T, src string) {
	t.Helper()
	other := t.TempDir()
	_ = os.Setenv("KRNR_HOME", other)
	// init empty DB so importer writes into it
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB for fresh env: %v", err)
	}
	_ = dbConn.Close()
	rootCmd.SetArgs([]string{"import", "set", src})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import set CLI failed: %v", err)
	}
}

func TestExportSetAndImportIntoFreshDB(t *testing.T) {
	// export via repository to avoid CLI state
	dst := exportSetFile(t, "expset", []string{"echo HELLO"})
	importIntoFreshHome(t, dst)

	// verify it exists and can be run
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB after import: %v", err)
	}
	defer func() { _ = dbConn2.Close() }()
	r := registry.NewRepository(dbConn2)
	cs, err := r.GetCommandSetByName("expset")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected imported set expset to exist")
	}
}

func createExistingSetInCurrentHome(t *testing.T, name string, cmds []string) {
	t.Helper()
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB dest: %v", err)
	}
	r2 := registry.NewRepository(dbConn2)
	desc2 := name + "-existing"
	id2, err := r2.CreateCommandSet(name, &desc2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet existing: %v", err)
	}
	for i, c := range cmds {
		if _, err := r2.AddCommand(id2, i+1, c); err != nil {
			t.Fatalf("AddCommand existing: %v", err)
		}
	}
	_ = dbConn2.Close()
}

func verifyCommands(t *testing.T, name string, expected []string) {
	t.Helper()
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB dest: %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	cs, err := r.GetCommandSetByName(name)
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected %s after import", name)
	}
	if len(cs.Commands) != len(expected) {
		t.Fatalf("unexpected merged commands length: %+v", cs.Commands)
	}
	for i, e := range expected {
		if cs.Commands[i].Command != e {
			t.Fatalf("unexpected merged command at %d: got %s want %s", i, cs.Commands[i].Command, e)
		}
	}
}

func TestImportSetCliMerge(t *testing.T) {
	src := setupTempDB(t)
	dbConnSrc, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB src: %v", err)
	}
	r := registry.NewRepository(dbConnSrc)
	desc := "cli-merge-src"
	id, err := r.CreateCommandSet("cli-merge", &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet src: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo A"); err != nil {
		t.Fatalf("AddCommand src: %v", err)
	}
	dst := filepath.Join(src, "cli-merge.db")
	if err := exporter.ExportCommandSet(dbConnSrc, "cli-merge", dst); err != nil {
		t.Fatalf("export set src failed: %v", err)
	}
	_ = dbConnSrc.Close()

	_ = setupTempDB(t)
	createExistingSetInCurrentHome(t, "cli-merge", []string{"echo B", "echo C"})

	rootCmd.SetArgs([]string{"import", "set", dst, "--on-conflict=merge", "--dedupe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import set merge failed: %v", err)
	}

	verifyCommands(t, "cli-merge", []string{"echo B", "echo C", "echo A"})
}

func TestImportInteractiveSetMerge(t *testing.T) {
	// create source and export
	dst := exportSetFile(t, "interactive-merge", []string{"echo A"})

	// create dest with overlapping commands
	_ = setupTempDB(t)
	createExistingSetInCurrentHome(t, "interactive-merge", []string{"echo B", "echo C"})

	// run interactive import: choose 2 (set), path, on-conflict=merge, dedupe=y
	input := dst + "\nmerge\ny\n"
	rootCmd.SetArgs([]string{"import"})
	rootCmd.SetIn(strings.NewReader("2\n" + input))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("interactive import failed: %v", err)
	}

	verifyCommands(t, "interactive-merge", []string{"echo B", "echo C", "echo A"})
}

func exportDBWithSet(t *testing.T, setName string, cmds []string) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("KRNR_HOME", tmp)
	rootCmd.SetArgs([]string{"save", setName, "-c"})
	// Use repo to add commands for determinism
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	r := registry.NewRepository(dbConn)
	desc := setName
	id, err := r.CreateCommandSet(setName, &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	for i, c := range cmds {
		if _, err := r.AddCommand(id, i+1, c); err != nil {
			t.Fatalf("AddCommand: %v", err)
		}
	}
	_ = dbConn.Close()
	dst := filepath.Join(tmp, "fulldb.db")
	if err := exporter.ExportDatabase(dst); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	return dst, tmp
}

func TestExportDatabaseAndImportOverwrite(t *testing.T) {
	// Save a set and export DB
	dst, tmp := exportDBWithSet(t, "dbset", []string{"echo DB"})

	// Test CLI import overwrite behavior
	tmpOther := t.TempDir()
	testImportDBOverwrite(t, dst, tmpOther)

	// verify set exists in imported DB
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB after import: %v", err)
	}
	defer func() { _ = dbConn2.Close() }()
	r := registry.NewRepository(dbConn2)
	cs, err := r.GetCommandSetByName("dbset")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected imported DB to contain dbset")
	}
	// import should have created a file which can be opened and contains the set
	// also test using importer functions directly for sanity
	db2 := filepath.Join(tmp, "sanity.db")
	if err := exporter.ExportDatabase(db2); err != nil {
		t.Fatalf("sanity export failed: %v", err)
	}
	// import into other
	if err := importer.ImportDatabase(db2, true, importer.ImportOptions{}); err != nil {
		t.Fatalf("import db direct failed: %v", err)
	}
	// check presence
	checkImportedSet(t, "dbset")
}

func testImportDBOverwrite(t *testing.T, src, other string) {
	t.Helper()
	_ = os.Setenv("KRNR_HOME", other)
	// create a destination DB to cause conflict
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	_ = dbConn.Close()
	// import without overwrite should fail
	_ = importDbCmd.Flags().Set("overwrite", "false")
	rootCmd.SetArgs([]string{"import", "db", src})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected import db to fail when dst exists and overwrite not set")
	}
	// import with overwrite should succeed
	rootCmd.SetArgs([]string{"import", "db", src, "--overwrite"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import db with overwrite failed: %v", err)
	}
}

func checkImportedSet(t *testing.T, name string) {
	t.Helper()
	dbConn3, _ := db.InitDB()
	defer func() { _ = dbConn3.Close() }()
	r2 := registry.NewRepository(dbConn3)
	cs2, _ := r2.GetCommandSetByName(name)
	if cs2 == nil {
		t.Fatalf("expected %s present after direct import", name)
	}
}
