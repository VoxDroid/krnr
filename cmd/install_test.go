package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/VoxDroid/krnr/internal/install"
)

func TestInstallDryRunCLI(t *testing.T) {
	tmp := t.TempDir()
	// create dummy src
	src := filepath.Join(tmp, "srcfile")
	_ = os.WriteFile(src, []byte("exe"), 0o644)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", oldHome)

	oldArgs := rootCmd.Args
	_ = oldArgs

	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd.SetArgs([]string{"install", "--dry-run", "--from", src})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	wOut.Close()
	var buf bytes.Buffer
	io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("Planned actions for install")) {
		t.Fatalf("expected planned actions, got: %s", out)
	}
}

func TestPlanInstallUserDir(t *testing.T) {
	p := install.DefaultUserBin()
	if p == "" {
		t.Fatalf("expected default user bin")
	}
}
