package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func setupRollbackRepoCLI(t *testing.T) *registry.Repository {
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
	_ = r.DeleteCommandSet("rcli")
	desc := "rollback cli test"
	id, err := r.CreateCommandSet("rcli", &desc, nil, nil, nil)
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

func TestRollbackCLIToNonEmptyVersion(t *testing.T) {
	r := setupRollbackRepoCLI(t)
	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// run rollback CLI to version 2
	rootCmd.SetArgs([]string{"rollback", "rcli", "--version", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rollback CLI failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if out == "" {
		t.Fatalf("expected output from rollback command, got empty")
	}

	// run describe CLI to verify commands
	oldOut = os.Stdout
	rOut, wOut, _ = os.Pipe()
	os.Stdout = wOut
	rootCmd.SetArgs([]string{"describe", "rcli"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("describe CLI failed: %v", err)
	}

	_ = wOut.Close()
	buf.Reset()
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out = buf.String()
	if out == "" {
		t.Fatalf("expected describe output, got empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("1: a")) || !bytes.Contains(buf.Bytes(), []byte("2: b")) {
		t.Fatalf("describe output did not include rolled back commands: %s", out)
	}

	// verify via repo directly
	cs, err := r.GetCommandSetByName("rcli")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands after rollback to version 2, got %d", len(cs.Commands))
	}
}
