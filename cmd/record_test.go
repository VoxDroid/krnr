package cmd

import (
	"bytes"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestRecordCommand_SavesCommandsFromStdin(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	// prepare stdin with two commands and an empty line + comment
	input := "echo record1\n# comment\necho record2\n"
	// inject input into the cobra command's input reader
	recordCmd.SetIn(bytes.NewBufferString(input))

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("record-test")

	if err := recordCmd.RunE(recordCmd, []string{"record-test"}); err != nil {
		t.Fatalf("recordCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("record-test")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set")
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "echo record1" || cs.Commands[1].Command != "echo record2" {
		t.Fatalf("unexpected commands: %+v", cs.Commands)
	}

	// cleanup
	_ = r.DeleteCommandSet("record-test")
}
