package registry

import (
	"strconv"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestVersions_WindowsPathAndUnicode(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)
	_ = r.DeleteCommandSet("win-α")
	desc := "win path and unicode test"
	id, err := r.CreateCommandSet("win-α", &desc, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	cmds1 := []string{`echo "C:\\Program Files\\MyApp\\bin"`, `powershell -EncodedCommand SGVsbG8=`}
	if err := r.ReplaceCommands(id, cmds1); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}

	// unicode command
	cmds2 := []string{"echo こんにちは"}
	if err := r.ReplaceCommands(id, cmds2); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	// verify versions exist and commands preserved
	vers, err := r.ListVersions(id)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(vers) < 3 {
		t.Fatalf("expected at least 3 versions, got %d", len(vers))
	}

	// apply first non-empty update (version 2) and verify commands match cmds1
	if err := r.ApplyVersionByName("win-α", 2); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}
	cs, err := r.GetCommandSetByName("win-α")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if len(cs.Commands) != len(cmds1) {
		t.Fatalf("expected %d commands after apply, got %d", len(cmds1), len(cs.Commands))
	}
	for i := range cmds1 {
		if cs.Commands[i].Command != cmds1[i] {
			t.Fatalf("command mismatch at %d: expected %q got %q", i, cmds1[i], cs.Commands[i].Command)
		}
	}
}

func TestVersions_LongHistory(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)
	_ = r.DeleteCommandSet("longhist")
	desc := "long history test"
	id, err := r.CreateCommandSet("longhist", &desc, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// create many versions
	const n = 60
	for i := 1; i <= n; i++ {
		cmd := []string{"echo v" + strconv.Itoa(i)}
		if err := r.ReplaceCommands(id, cmd); err != nil {
			t.Fatalf("ReplaceCommands %d: %v", i, err)
		}
	}
	vers, err := r.ListVersions(id)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	// initial create + n updates = n+1 versions
	if len(vers) < n+1 {
		t.Fatalf("expected at least %d versions, got %d", n+1, len(vers))
	}
	// newest version number should be n+1
	if vers[0].Version != n+1 {
		t.Fatalf("expected newest version %d, got %d", n+1, vers[0].Version)
	}
}
