package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func exportedDBDefaultPath(t *testing.T, tmp string) string {
	t.Helper()
	date := time.Now().UTC().Format("2006-01-02")
	files, _ := os.ReadDir(tmp)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "krnr-"+date) && strings.HasSuffix(f.Name(), ".db") {
			return filepath.Join(tmp, f.Name())
		}
	}
	t.Fatalf("expected exported DB with prefix krnr-%s.db, files: %+v", date, files)
	return ""
}

func TestExportDbDefaultDstCreatesFile(t *testing.T) {
	// switch to a temp dir so default file is created here
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(wd) }()

	// create a DB with some data
	_ = os.Setenv("KRNR_HOME", tmp)
	rootCmd.SetArgs([]string{"save", "xset", "-c", "echo X"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// run export without --dst
	rootCmd.SetArgs([]string{"export", "db"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export db failed: %v", err)
	}

	// find exported DB path
	exp := exportedDBDefaultPath(t, tmp)

	// sanity: import into fresh and verify
	other := t.TempDir()
	importAndVerify(t, exp, other, "xset")
}

func importAndVerify(t *testing.T, exp, other, name string) {
	t.Helper()
	_ = os.Setenv("KRNR_HOME", other)
	// init to create file then close
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	_ = dbConn.Close()

	rootCmd.SetArgs([]string{"import", "db", exp, "--overwrite"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import db default failed: %v", err)
	}
	// verify set exists in imported
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB after import: %v", err)
	}
	defer func() { _ = dbConn2.Close() }()
	r := registry.NewRepository(dbConn2)
	cs, err := r.GetCommandSetByName(name)
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected %s after import", name)
	}
}
