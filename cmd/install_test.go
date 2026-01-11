package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/install"
)

func TestInstallDryRunCLI(t *testing.T) {
	tmp := t.TempDir()
	// create dummy src
	src := filepath.Join(tmp, "srcfile")
	_ = os.WriteFile(src, []byte("exe"), 0o644)

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
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

func TestSystemInstall_PromptsToAddToPath(t *testing.T) {
	// Use a custom path to ensure it's not on PATH
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, "krnr")
	_ = os.MkdirAll(installDir, 0o755)
	// create dummy src
	src := filepath.Join(tmp, "srcfile")
	_ = os.WriteFile(src, []byte("exe"), 0o644)

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Capture stdout and provide interactive responses (no, then no)
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	oldIn := os.Stdin
	inReader, inWriter, _ := os.Pipe()
	_, _ = inWriter.Write([]byte("n\n")) // answer 'n' to add-to-path prompt
	_ = inWriter.Close()
	os.Stdin = inReader
	defer func() { os.Stdin = oldIn }()

	rootCmd.SetArgs([]string{"install", "--system", "--path", installDir, "--from", src})
	_ = rootCmd.Execute()

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	if strings.Contains(out, "Target dir is not on PATH") || (strings.Contains(out, "Add ") && strings.Contains(out, "PATH")) {
		return
	}
	t.Fatalf("expected prompt or plan suggesting adding to PATH, got: %s", out)
}
