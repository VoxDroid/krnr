package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestHistoryCLI_UnicodeAndLong(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("unicöde")
	d := "unicode history"
	id, err := r.CreateCommandSet("unicöde", &d, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"echo あ"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"echo い"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd.SetArgs([]string{"history", "unicöde"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("history CLI failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if out == "" {
		t.Fatalf("expected history output, got empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("v1")) || !bytes.Contains(buf.Bytes(), []byte("v2")) {
		t.Fatalf("expected versions v1 and v2 in output, got: %s", out)
	}
}
