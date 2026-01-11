package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func setupHistoryRepo(t *testing.T) *registry.Repository {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	// cleanup env
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })

	// init DB and repo
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	// cleanup DB
	t.Cleanup(func() { _ = dbConn.Close() })
	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("hst")
	desc := "history test"
	id, err := r.CreateCommandSet("hst", &desc, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"a", "b"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"c"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}
	return r
}

func TestHistoryCLIOutputsSomething(t *testing.T) {
	_ = setupHistoryRepo(t)
	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// run history CLI
	rootCmd.SetArgs([]string{"history", "hst"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("history CLI failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if out == "" {
		t.Fatalf("expected output from history command, got empty")
	}
}

func TestRollbackCLIAppliesVersion(t *testing.T) {
	r := setupHistoryRepo(t)
	// run rollback CLI to version 1
	rootCmd.SetArgs([]string{"rollback", "hst", "--version", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rollback CLI failed: %v", err)
	}
	// Verify rollback applied
	cs, err := r.GetCommandSetByName("hst")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set post-rollback")
	}
	if len(cs.Commands) != 0 {
		t.Fatalf("expected 0 commands after rollback to version 1, got %d", len(cs.Commands))
	}
}
