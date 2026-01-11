package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestTagAddListAndFilter(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// create a command set
	rootCmd.SetArgs([]string{"save", "s1", "-c", "echo 1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// add a tag
	rootCmd.SetArgs([]string{"tag", "add", "s1", "t1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tag add failed: %v", err)
	}

	// list tags for s1
	rootCmd.SetArgs([]string{"tag", "list", "s1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tag list failed: %v", err)
	}

	// list with tag filter
	rootCmd.SetArgs([]string{"list", "--tag", "t1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list --tag failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("t1")) {
		t.Fatalf("expected tag output, got: %s", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("s1")) {
		t.Fatalf("expected list to contain s1, got: %s", out)
	}
}