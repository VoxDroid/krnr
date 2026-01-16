package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestUpdateCommandSetAndReplaceCommandsRecordsSingleUpdate(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)
	_ = r.DeleteCommandSet("upvtest")
	desc := "up test"
	id, err := r.CreateCommandSet("upvtest", &desc, nil, nil, []string{"a"})
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// perform combined update
	if err := r.UpdateCommandSetAndReplaceCommands(id, "upvtest", &desc, nil, nil, []string{"x"}, []string{"alpha", "beta"}); err != nil {
		t.Fatalf("UpdateCommandSetAndReplaceCommands: %v", err)
	}

	vers, err := r.ListVersions(id)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	// expect exactly two versions: create + 1 update
	if len(vers) != 2 {
		t.Fatalf("expected exactly 2 versions after update, got %d", len(vers))
	}
	if vers[0].Operation != "update" {
		t.Fatalf("expected newest operation 'update', got %s", vers[0].Operation)
	}
	if len(vers[0].Commands) != 2 || vers[0].Commands[0] != "alpha" {
		t.Fatalf("expected updated commands in version, got %+v", vers[0].Commands)
	}
}
