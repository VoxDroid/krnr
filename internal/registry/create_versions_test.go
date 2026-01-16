package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestCreateRecordsSingleVersion(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)
	// cleanup
	_ = r.DeleteCommandSet("create-vtest")
	desc := "create version test"
	// create with initial commands
	id, err := r.CreateCommandSet("create-vtest", &desc, nil, nil, []string{"echo hi"})
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	vers, err := r.ListVersions(id)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(vers) != 1 {
		t.Fatalf("expected exactly 1 version after create, got %d", len(vers))
	}
	if vers[0].Operation != "create" {
		t.Fatalf("expected operation 'create', got %s", vers[0].Operation)
	}
	if len(vers[0].Commands) != 1 || vers[0].Commands[0] != "echo hi" {
		t.Fatalf("expected initial command present in version, got %+v", vers[0].Commands)
	}
}
