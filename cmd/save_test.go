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
	local.Flags().StringArrayP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	local.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	local.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	if err := local.Flags().Set("command", "echo save1"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := local.Flags().Set("command", "echo save2"); err != nil {
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

func runSaveWithInput(t *testing.T, initialName string, stdin string, commands []string) *registry.Repository {
	t.Helper()
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	// do not close DB: caller test handles environment between tests
	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet(initialName)
	_ = r.DeleteCommandSet(initialName + "-2")
	// create a pre-existing set
	if _, err := r.CreateCommandSet(initialName, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	local := &cobra.Command{RunE: saveCmd.RunE, Args: saveCmd.Args}
	local.Flags().StringP("description", "d", "", "Description for the command set")
	local.Flags().StringArrayP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	local.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	local.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	local.SetIn(bytes.NewBufferString(stdin))
	for _, c := range commands {
		if err := local.Flags().Set("command", c); err != nil {
			t.Fatalf("set flag: %v", err)
		}
	}

	if err := local.RunE(local, []string{initialName}); err != nil {
		t.Fatalf("saveCmd failed: %v", err)
	}
	return r
}

func TestSaveCommand_PromptsOnDuplicateName(t *testing.T) {
	r := runSaveWithInput(t, "save-dup", "save-dup-2\n", []string{"echo a", "echo b"})
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
	assertCommandsPresent(t, cs, []string{"echo a", "echo b"})

	// cleanup
	_ = r.DeleteCommandSet("save-dup")
	_ = r.DeleteCommandSet("save-dup-2")
}

func assertCommandsPresent(t *testing.T, cs *registry.CommandSet, expected []string) {
	t.Helper()
	if cs == nil {
		t.Fatalf("nil CommandSet")
	}
	for _, e := range expected {
		found := false
		for _, c := range cs.Commands {
			if c.Command == e {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected command %s to be present in %+v", e, cs.Commands)
		}
	}
}
