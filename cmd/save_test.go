package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestSaveCommand_SavesCommands(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("save-test")

	// use a fresh command with its own FlagSet to avoid global flag state
	local := &cobra.Command{RunE: saveCmd.RunE, Args: saveCmd.Args}
	local.Flags().StringP("description", "d", "", "Description for the command set")
	local.Flags().StringSliceP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	local.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	local.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	if err := local.Flags().Set("command", "echo save1,echo save2"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	if err := local.RunE(local, []string{"save-test"}); err != nil {
		t.Fatalf("saveCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("save-test")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set")
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "echo save1" || cs.Commands[1].Command != "echo save2" {
		t.Fatalf("unexpected commands: %+v", cs.Commands)
	}

	// cleanup
	_ = r.DeleteCommandSet("save-test")
}

func TestSaveCommand_PromptsOnDuplicateName(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("save-dup")
	_ = r.DeleteCommandSet("save-dup-2")
	// create a pre-existing set
	if _, err := r.CreateCommandSet("save-dup", nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// prepare input: new name
	local := &cobra.Command{RunE: saveCmd.RunE, Args: saveCmd.Args}
	local.Flags().StringP("description", "d", "", "Description for the command set")
	local.Flags().StringSliceP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	local.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	local.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	local.SetIn(bytes.NewBufferString("save-dup-2\n"))
	if err := local.Flags().Set("command", "echo a,echo b"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	if err := local.RunE(local, []string{"save-dup"}); err != nil {
		t.Fatalf("saveCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("save-dup-2")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set created with new name")
	}
	if len(cs.Commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(cs.Commands))
	}
	// ensure provided commands are present
	foundA := false
	foundB := false
	for _, c := range cs.Commands {
		if c.Command == "echo a" {
			foundA = true
		}
		if c.Command == "echo b" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected commands 'echo a' and 'echo b' to be present, got %+v", cs.Commands)
	}

	// cleanup
	_ = r.DeleteCommandSet("save-dup")
	_ = r.DeleteCommandSet("save-dup-2")
}