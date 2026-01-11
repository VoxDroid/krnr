package cmd

import (
	"bytes"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestDeleteCommand_PromptsAndDeletesWhenConfirmed(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("del-test")
	if _, err := r.CreateCommandSet("del-test", nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// simulate user typing 'y' to confirm
	deleteCmd.SetIn(bytes.NewBufferString("y\n"))
	if err := deleteCmd.RunE(deleteCmd, []string{"del-test"}); err != nil {
		t.Fatalf("deleteCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("del-test")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs != nil {
		t.Fatalf("expected command set to be deleted")
	}
}

func TestDeleteCommand_AbortsOnNo(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("del-abort")
	if _, err := r.CreateCommandSet("del-abort", nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// simulate user typing 'n' to abort
	deleteCmd.SetIn(bytes.NewBufferString("n\n"))
	if err := deleteCmd.RunE(deleteCmd, []string{"del-abort"}); err != nil {
		t.Fatalf("deleteCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("del-abort")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set to still exist")
	}

	// cleanup
	_ = r.DeleteCommandSet("del-abort")
}

func TestDeleteCommand_SkipsPromptWithYesFlag(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("del-yes")
	if _, err := r.CreateCommandSet("del-yes", nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// set the yes flag to skip prompting
	_ = deleteCmd.Flags().Set("yes", "true")
	defer func() { _ = deleteCmd.Flags().Set("yes", "false") }()

	if err := deleteCmd.RunE(deleteCmd, []string{"del-yes"}); err != nil {
		t.Fatalf("deleteCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("del-yes")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs != nil {
		t.Fatalf("expected command set to be deleted when --yes is provided")
	}
}
