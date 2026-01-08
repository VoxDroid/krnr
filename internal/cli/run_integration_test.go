package cli

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestRunIntegrationDryRun(t *testing.T) {
	// Set HOME to tempdir so DB is isolated
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	desc := "integration"
	id, err := r.CreateCommandSet("int-set", &desc, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo one"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}
	if _, err := r.AddCommand(id, 2, "echo two"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	// Run with dry-run and capture output
	var out bytes.Buffer
	var errb bytes.Buffer
	e := &executor.Executor{DryRun: true, Verbose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cs, err := r.GetCommandSetByName("int-set")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}

	for _, c := range cs.Commands {
		if err := e.Execute(ctx, c.Command, "", &out, &errb); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	}

	if out.Len() == 0 {
		t.Fatalf("expected output from dry-run, got empty")
	}
	if errb.Len() != 0 {
		t.Fatalf("expected no stderr for dry-run, got: %q", errb.String())
	}
}
