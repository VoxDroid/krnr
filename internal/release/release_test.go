package release

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestReleaseScript runs the release script in a temporary environment and verifies
// that artifacts and checksum files are generated. This is an _integration_ test
// and is intentionally minimal — it invokes `scripts/release.sh` with a small set
// of targets. The test is skipped on Windows because the CI Ubuntu runner is the
// canonical environment for release packaging.
func TestReleaseScript_RunCreatesDist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release script integration test skipped on Windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	version := "v0.0.0-e2e"
	script := filepath.Join("..", "..", "scripts", "release.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("release script not found: %v", err)
	}

	cmd := exec.CommandContext(ctx, "bash", script, version, "linux/amd64")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("release script failed: %v", err)
	}

	dist := filepath.Join("dist")
	if _, err := os.Stat(dist); err != nil {
		t.Fatalf("dist directory not created: %v", err)
	}

	// Expect at least one tar.gz or zip file
	found := false
	entries, err := os.ReadDir(dist)
	if err != nil {
		t.Fatalf("read dist dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tar.gz") || strings.HasSuffix(e.Name(), ".zip") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no release archives found in dist")
	}
}

func TestReleaseScript_SHAFileContainsVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release script integration test skipped on Windows")
	}
	version := "v0.0.0-e2e"
	dist := filepath.Join("dist")
	shaFile := filepath.Join(dist, "krnr-"+version+"-SHA256SUMS")
	b, err := os.ReadFile(shaFile)
	if err != nil {
		t.Fatalf("failed to read sha file: %v", err)
	}
	if !strings.Contains(string(b), version) {
		t.Fatalf("sha file does not mention version: %s", string(b))
	}
	// cleanup artifacts to keep workspace clean
	_ = os.RemoveAll(dist)
}
