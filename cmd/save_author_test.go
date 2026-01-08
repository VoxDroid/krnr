package cmd

import (
	"bytes"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/user"
)

func TestSaveUsesStoredWhoami(t *testing.T) {
	// set stored whoami
	_ = user.ClearProfile()
	if err := user.SetProfile(user.Profile{Name: "Carol", Email: "carol@example.com"}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer dbConn.Close()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("whoami-save")

	// run save without author flag
	buf := bytes.Buffer{}
	saveCmd.SetOut(&buf)
	// set command flag and execute save
	saveCmd.Flags().Set("command", "echo ok")
	if err := saveCmd.RunE(saveCmd, []string{"whoami-save"}); err != nil {
		t.Fatalf("saveCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("whoami-save")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set")
	}
	if !cs.AuthorName.Valid || cs.AuthorName.String != "Carol" {
		t.Fatalf("expected author Carol, got %+v", cs.AuthorName)
	}
	if !cs.AuthorEmail.Valid || cs.AuthorEmail.String != "carol@example.com" {
		t.Fatalf("expected author email carol@example.com, got %+v", cs.AuthorEmail)
	}

	// cleanup
	_ = r.DeleteCommandSet("whoami-save")
	_ = user.ClearProfile()
}
