package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestRunWithParamsCLI(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// create a command set with parameter
	rootCmd.SetArgs([]string{"save", "greet", "-c", "echo Hello {{name}}", "-d", "greet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save greet failed: %v", err)
	}

	// run with param
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd.SetArgs([]string{"run", "greet", "--param", "name=world", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run greet failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("Hello world")) {
		t.Fatalf("expected Hello world, got: %s", out)
	}
}
