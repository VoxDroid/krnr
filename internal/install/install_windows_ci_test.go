//go:build windows
// +build windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallAndUninstall_CI exercises install/uninstall logic on Windows runners
// without modifying the actual system PATH (uses KRNR_TEST_NO_SETX test mode).
func TestInstallAndUninstall_CI(t *testing.T) {
	// Prevent persistent modifications to PATH in CI
	t.Setenv("KRNR_TEST_NO_SETX", "1")
	// Isolate data dir and home directory to temp
	tmp := t.TempDir()
	t.Setenv("KRNR_HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	// Create a fake source 'executable'
	src := filepath.Join(tmp, "krnr.exe")
	if err := os.WriteFile(src, []byte("exe"), 0o644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}

	opts := Options{From: src, AddToPath: true}
	actions, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Expect at least one action mentioning PATH/Add
	foundAdd := false
	for _, a := range actions {
		if strings.Contains(strings.ToLower(a), "add") || strings.Contains(strings.ToLower(a), "path") {
			foundAdd = true
			break
		}
	}
	if !foundAdd {
		t.Fatalf("expected install actions to mention adding to PATH, got: %v", actions)
	}

	// Verify binary exists at expected user bin
	target := filepath.Join(DefaultUserBin(), "krnr.exe")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("binary not found at %s: %v", target, err)
	}

	// Run uninstall (should not actually call setx because of KRNR_TEST_NO_SETX)
	uActions, err := Uninstall(false)
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	// We expect uninstall to either note a test-mode removal or perform removal messages
	noteFound := false
	for _, a := range uActions {
		al := strings.ToLower(a)
		if strings.Contains(al, "note: would remove") || strings.Contains(al, "removed") || strings.Contains(al, "restore") {
			noteFound = true
			break
		}
	}
	if !noteFound {
		t.Fatalf("unexpected uninstall actions: %v", uActions)
	}
}
