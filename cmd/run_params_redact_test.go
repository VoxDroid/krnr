package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestRunDryRunRedactsSecretsCLI(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// prepare DB and repo
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("secret-test")
	desc := "secret param test"
	id, err := r.CreateCommandSet("secret-test", &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo SECRET {{token}}"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	// set env var to simulate env: binding
	_ = os.Setenv("MY_TOKEN", "supersecret")

	// capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd.SetArgs([]string{"run", "secret-test", "--param", "token=env:MY_TOKEN", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if bytes.Contains(buf.Bytes(), []byte("supersecret")) {
		t.Fatalf("expected secret to be redacted in output, but found it: %s", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("<redacted>")) {
		t.Fatalf("expected redacted placeholder in output, got: %s", out)
	}
}
