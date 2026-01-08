package ci

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// helper to find the repo-root-local scripts/lint.sh path
func findLintScript(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		candidate := filepath.Join(cwd, "scripts", "lint.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	t.Fatalf("scripts/lint.sh not found in repository tree")
	return ""
}

// helper to run the lint script with a modified PATH and return output and exit error
func runLintWithPath(t *testing.T, path string) (string, error) {
	if runtime.GOOS == "windows" {
		t.Skip("script tests skipped on Windows")
	}
	script := findLintScript(t)
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "PATH="+path)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestLintScript_NoLocalNoDocker(t *testing.T) {
	tmp := t.TempDir()
	out, err := runLintWithPath(t, tmp)
	if err != nil {
		// script exits zero in this scenario; treat non-zero as failure
		t.Fatalf("script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "golangci-lint not found locally") {
		t.Fatalf("expected golangci-lint-not-found message, got: %s", out)
	}
	if !strings.Contains(out, "Docker not available; cannot run docker fallback") {
		t.Fatalf("expected docker-not-available message, got: %s", out)
	}
}

func TestLintScript_LocalError_NoDocker(t *testing.T) {
	tmp := t.TempDir()
	// create fake golangci-lint that prints export-data error and exits 1
	script := "#!/bin/sh\necho 'error: unsupported version' >&2\nexit 1\n"
	bin := filepath.Join(tmp, "golangci-lint")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake linter: %v", err)
	}

	out, err := runLintWithPath(t, tmp)
	if err != nil {
		t.Fatalf("script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "unsupported version") && !strings.Contains(out, "could not load export data") {
		t.Fatalf("expected export-data message, got: %s", out)
	}
	if !strings.Contains(out, "Docker not available; cannot run docker fallback") {
		t.Fatalf("expected docker-not-available message, got: %s", out)
	}
}

func TestLintScript_DockerFallbackSucceeds(t *testing.T) {
	tmp := t.TempDir()
	// fake golangci-lint that fails with export-data error
	lscript := "#!/bin/sh\necho 'error: unsupported version' >&2\nexit 1\n"
	lbin := filepath.Join(tmp, "golangci-lint")
	if err := os.WriteFile(lbin, []byte(lscript), 0o755); err != nil {
		t.Fatalf("write fake linter: %v", err)
	}
	// fake docker that prints a message and exits 0
	dscript := "#!/bin/sh\necho 'mock docker called'\nexit 0\n"
	dbin := filepath.Join(tmp, "docker")
	if err := os.WriteFile(dbin, []byte(dscript), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	out, err := runLintWithPath(t, tmp)
	if err != nil {
		t.Fatalf("script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Attempting Docker-based golangci-lint") {
		t.Fatalf("expected Docker fallback attempt, got: %s", out)
	}
	if !strings.Contains(out, "mock docker called") {
		t.Fatalf("expected mock docker to be invoked, got: %s", out)
	}
}
