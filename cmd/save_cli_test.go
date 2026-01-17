package cmd

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/spf13/cobra"
)

// TestSaveWithSplitArgs simulates a common shell quoting mistake where a
// command with embedded quotes gets split into multiple positional args.
// We expect the CLI to join leftover positional args into a single command
// and save it.
func TestSaveWithSplitArgs(t *testing.T) {
	_ = setupTempDB(t)
	// Simulate: krnr save CMD systeminfo | findstr /C:OS Name /C:OS Version
	rootCmd.SetArgs([]string{"save", "CMD", "systeminfo", "|", "findstr", "/C:OS", "Name", "/C:OS", "Version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Verify the command set exists in the DB and contains the joined command
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	cs, err := r.GetCommandSetByName("CMD")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected saved command set 'CMD' in DB")
	}
	found := false
	joined := "systeminfo | findstr /C:OS Name /C:OS Version"
	for _, c := range cs.Commands {
		if c.Command == joined {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected joined command %q in commands, got: %+v", joined, cs.Commands)
	}
}

func TestSaveWithSplitArgs_MergeIntoSingle(t *testing.T) {
	_ = setupTempDB(t)
	// Simulate the common broken parse where the -c flag ends up containing a
	// truncated command and the leftover tokens are positional args; emulate
	// the flag parser returning a single command element and the CLI args
	// holding the remaining tokens.
	local := &cobra.Command{RunE: saveCmd.RunE, Args: saveCmd.Args}
	local.Flags().StringArrayP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	if err := local.Flags().Set("command", "systeminfo | findstr /C:"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	// Run with leftover tokens that should be merged into the last command
	if err := local.RunE(local, []string{"CMDS", "OS", "Name", "/C:OS", "Version"}); err != nil {
		t.Fatalf("save RunE failed: %v", err)
	}

	// Verify DB had a single merged command
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	cs, err := r.GetCommandSetByName("CMDS")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected saved command set 'CMDS' in DB")
	}
	if len(cs.Commands) != 1 {
		t.Fatalf("expected single merged command, got: %+v", cs.Commands)
	}
	expected := "systeminfo | findstr /C:\"OS Name\" /C:\"OS Version\""
	if cs.Commands[0].Command != expected {
		t.Fatalf("expected merged and normalized command %q, got %q", expected, cs.Commands[0].Command)
	}
}

func TestSaveCommand_PreservesCommasInFlag(t *testing.T) {
	_ = setupTempDB(t)
	// Ensure no pre-existing set with the same name
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r2 := registry.NewRepository(dbConn)
	_ = r2.DeleteCommandSet("PS-preserve-comma")

	// Simulate: krnr save PS-preserve-comma -c "Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture"
	rootCmd.SetArgs([]string{"save", "PS-preserve-comma", "-c", "Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify the command set exists and the command was preserved as a single entry
	cs, err := r2.GetCommandSetByName("PS-preserve-comma")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected saved command set 'PS-preserve-comma' in DB")
	}
	expected := "Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture"
	found := false
	for _, c := range cs.Commands {
		if c.Command == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected preserved command %q to be present in commands, got: %+v", expected, cs.Commands)
	}
}
