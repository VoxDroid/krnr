//go:build windows
// +build windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper used by Windows CI tests to install a fake executable and return temp dir and actions.
func setupWindowsCIInstall(t *testing.T) (tmp string, actions []string) {
	t.Setenv("KRNR_TEST_NO_SETX", "1")
	tmp = t.TempDir()
	t.Setenv("KRNR_HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	src := filepath.Join(tmp, "krnr.exe")
	if err := os.WriteFile(src, []byte("exe"), 0o644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}
	opts := Options{From: src, AddToPath: true}
	actions, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	return tmp, actions
}

func TestInstallAndUninstall_CI_InstallActions(t *testing.T) {
	_, actions := setupWindowsCIInstall(t)
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
}

func TestInstallAndUninstall_CI_BinaryExists(t *testing.T) {
	_, _ = setupWindowsCIInstall(t)
	// Verify binary exists at expected user bin
	target := filepath.Join(DefaultUserBin(), "krnr.exe")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("binary not found at %s: %v", target, err)
	}
	// cleanup
	_, _ = Uninstall(false)
}

func TestInstallAndUninstall_CI_UninstallNotes(t *testing.T) {
	_, _ = setupWindowsCIInstall(t)
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
