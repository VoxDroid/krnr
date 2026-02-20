package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"os/user"
)

func TestDefaultUserBin(t *testing.T) {
	p := DefaultUserBin()
	if p == "" {
		t.Fatalf("expected a default user bin, got empty")
	}
	// Should be an absolute path
	if !filepath.IsAbs(p) {
		t.Fatalf("expected absolute path, got: %s", p)
	}
}

func TestPlanInstallDryRun(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{User: true, Path: tmp, From: filepath.Join(tmp, "src"), DryRun: true}
	// create dummy src
	_ = os.WriteFile(opts.From, []byte("hi"), 0o644)
	actions, target, err := PlanInstall(opts)
	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("expected actions, got none")
	}
	if filepath.Join(tmp, "krnr") != target && (runtime.GOOS != "windows" || filepath.Join(tmp, "krnr.exe") != target) {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestExecuteInstallCopiesFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: true, Path: tmp, From: src, DryRun: false}
	actions, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("expected actions")
	}
	// Ensure file exists
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	target := filepath.Join(tmp, binName)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target file, stat failed: %v", err)
	}
}

func TestInstallAddToPath(t *testing.T) {
	// Simulate a shell rc in a temporary HOME
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Make a .bashrc
	rc := filepath.Join(tmp, ".bashrc")
	_ = os.WriteFile(rc, []byte("# existing\n"), 0o644)

	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: true, Path: tmp, From: src, DryRun: false, AddToPath: true}
	// On Windows, enable test-mode BEFORE calling ExecuteInstall so we don't invoke PowerShell
	if runtime.GOOS == "windows" {
		_ = os.Setenv("KRNR_TEST_NO_SETX", "1")
		defer func() { _ = os.Unsetenv("KRNR_TEST_NO_SETX") }()
	}
	_, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall add-to-path: %v", err)
	}
	if runtime.GOOS == "windows" {
		m, err := loadMetadata()
		if err != nil {
			t.Fatalf("expected metadata on Windows: %v", err)
		}
		if m.TargetPath == "" {
			t.Fatalf("expected metadata target path set")
		}
		if m.PathFile != "UserEnv" {
			t.Fatalf("expected PathFile to be UserEnv, got: %s", m.PathFile)
		}
	} else {
		// check .bashrc contains our added line
		b, _ := os.ReadFile(rc)
		if !strings.Contains(string(b), "krnr") {
			t.Fatalf("expected krnr line in rc, got: %s", string(b))
		}
	}
}

func TestStatusReflectsMetadataAddedToPath(t *testing.T) {
	// Simulate a user install with AddToPath and ensure Status reflects the recorded metadata
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// create an rc so addToPathUnix will write to it
	rc := filepath.Join(tmp, ".bashrc")
	_ = os.WriteFile(rc, []byte("# existing\n"), 0o644)

	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: true, Path: filepath.Join(tmp, "krnr", "bin"), From: src, DryRun: false, AddToPath: true}
	if runtime.GOOS == "windows" {
		_ = os.Setenv("KRNR_TEST_NO_SETX", "1")
		defer func() { _ = os.Unsetenv("KRNR_TEST_NO_SETX") }()
	}
	_, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall add-to-path: %v", err)
	}

	// Ensure current process PATH does NOT include the new user dir (simulate shell not reloaded)
	t.Setenv("PATH", "")
	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !st.MetadataFound {
		t.Fatalf("expected metadata found")
	}
	if !st.UserOnPath {
		t.Fatalf("expected UserOnPath true from metadata: %+v", st)
	}
	if !st.UserInstalled {
		t.Fatalf("expected UserInstalled true: %+v", st)
	}
}

func TestDefaultUserBinHonorsSudoUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style HOME/SUDO_USER semantics only")
	}
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	origHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	tmp := t.TempDir()
	// simulate running under sudo: HOME appears to be root's, but SUDO_USER is set
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("SUDO_USER", cur.Username)

	want := filepath.Join(cur.HomeDir, "krnr", "bin")
	got := DefaultUserBin()
	if got != want {
		t.Fatalf("DefaultUserBin with SUDO_USER: want %q got %q", want, got)
	}
}

func TestPlanInstallUsesSudoUserHomeAndStatusFromMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style HOME/SUDO_USER semantics only")
	}
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	// isolate metadata
	krnrHome := t.TempDir()
	_ = os.Setenv("KRNR_HOME", krnrHome)
	defer func() { _ = os.Unsetenv("KRNR_HOME") }()

	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", t.TempDir()) // simulate being root
	_ = os.Setenv("SUDO_USER", cur.Username)
	defer func() {
		_ = os.Setenv("HOME", origHome)
		_ = os.Unsetenv("SUDO_USER")
	}()

	opts := Options{User: true, DryRun: true}
	actions, target, err := PlanInstall(opts)
	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	if !strings.Contains(target, cur.HomeDir) {
		t.Fatalf("expected target to use SUDO_USER home; actions=%v target=%s", actions, target)
	}

	// simulate recorded metadata (added-to-path)
	if err := saveMetadata(target, true, filepath.Join(cur.HomeDir, ".bashrc"), ""); err != nil {
		t.Fatalf("saveMetadata: %v", err)
	}

	// clear PATH so current process doesn't report on-path from env
	_ = os.Setenv("PATH", "")
	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !st.MetadataFound || !st.UserOnPath {
		t.Fatalf("expected UserOnPath true from metadata under SUDO_USER: %+v", st)
	}
}

func TestExecuteInstallAsSudoWritesMetadataAndUninstallable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style HOME/SUDO_USER semantics only")
	}
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	// simulate running installer under sudo (HOME becomes root-like) but target
	// and metadata should land in the original user's data dir. Use the
	// KRNR_TEST_SUDO_HOME helper so tests don't touch the real home.
	tmpRootHome := t.TempDir()
	_ = os.Setenv("HOME", tmpRootHome)
	_ = os.Setenv("SUDO_USER", cur.Username)
	defer func() {
		_ = os.Unsetenv("SUDO_USER")
		_ = os.Setenv("HOME", os.Getenv("HOME"))
	}()

	// direct the SUDO user's home to an isolated tempdir for the test
	sudoHome := t.TempDir()
	_ = os.Setenv("KRNR_TEST_SUDO_HOME", sudoHome)
	// install target under the fake sudo user's home
	installDir := filepath.Join(sudoHome, "krnr", "bin")
	_ = os.MkdirAll(installDir, 0o755)

	src := filepath.Join(t.TempDir(), "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: true, Path: installDir, From: src, DryRun: false, AddToPath: true}
	_, err = ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall: %v", err)
	}

	// metadata must be present in the SUDO user's KRNR_HOME (test helper path)
	metaPath := filepath.Join(sudoHome, ".krnr", "install_metadata.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("expected metadata at %s: %v", metaPath, err)
	}
	var m metadata
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if m.TargetPath == "" || !strings.Contains(m.TargetPath, installDir) {
		t.Fatalf("unexpected metadata TargetPath: %v", m.TargetPath)
	}

	// Now simulate running as the normal user (unset SUDO_USER, point KRNR_HOME)
	_ = os.Setenv("KRNR_HOME", filepath.Join(sudoHome, ".krnr"))
	_ = os.Unsetenv("SUDO_USER")

	// Status should reflect metadata (UserOnPath true) even if PATH is empty
	_ = os.Setenv("PATH", "")
	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !st.MetadataFound || !st.UserOnPath {
		t.Fatalf("expected UserOnPath true from metadata after sudo install: %+v", st)
	}

	// Uninstall (as non-sudo) should remove the installed binary
	actions, err := Uninstall(false)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, "krnr")); err == nil {
		t.Fatalf("expected binary removed by uninstall")
	}
	if len(actions) == 0 {
		t.Fatalf("expected uninstall actions")
	}

	// Re-install as sudo again into the same user home
	_ = os.Setenv("HOME", tmpRootHome)
	_ = os.Setenv("SUDO_USER", cur.Username)
	// ensure KRNR_TEST_SUDO_HOME remains set to sudoHome
	_, err = ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall (second): %v", err)
	}

	// Status as the normal user should again reflect metadata-added-to-PATH
	_ = os.Unsetenv("SUDO_USER")
	_ = os.Setenv("KRNR_HOME", filepath.Join(sudoHome, ".krnr"))
	_ = os.Setenv("PATH", "")
	st2, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus after reinstall: %v", err)
	}
	if !st2.MetadataFound || !st2.UserOnPath {
		t.Fatalf("expected UserOnPath true after reinstall: %+v", st2)
	}
}

func TestInstallAddToPathAndUninstall(t *testing.T) {
	// Simulate a shell rc in a temporary HOME
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Make a .bashrc
	rc := filepath.Join(tmp, ".bashrc")
	_ = os.WriteFile(rc, []byte("# existing\n"), 0o644)

	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: true, Path: tmp, From: src, DryRun: false, AddToPath: true}
	// On Windows, enable test-mode BEFORE calling ExecuteInstall so we don't invoke PowerShell
	if runtime.GOOS == "windows" {
		_ = os.Setenv("KRNR_TEST_NO_SETX", "1")
		defer func() { _ = os.Unsetenv("KRNR_TEST_NO_SETX") }()
	}
	_, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall add-to-path: %v", err)
	}
	// Now uninstall
	actions, err := Uninstall(false)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if runtime.GOOS == "windows" {
		// On Windows we can't check rc edits; just ensure uninstall recorded an action
		if len(actions) == 0 {
			t.Fatalf("expected uninstall actions on Windows")
		}
	} else {
		// rc should no longer contain krnr lines
		b2, _ := os.ReadFile(rc)
		if strings.Contains(string(b2), "krnr") {
			t.Fatalf("expected krnr lines removed, got: %s", string(b2))
		}
	}
	// binary should be removed
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	if _, err := os.Stat(filepath.Join(tmp, binName)); err == nil {
		t.Fatalf("expected binary to be removed")
	}
	if len(actions) == 0 {
		t.Fatalf("expected uninstall actions")
	}
}

func TestSystemInstallAddToPathAndUninstall_WindowsOnly(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only system PATH test")
	}
	// Use KRNR_TEST_NO_SETX to avoid changing the real machine PATH
	_ = os.Setenv("KRNR_TEST_NO_SETX", "1")
	defer func() { _ = os.Unsetenv("KRNR_TEST_NO_SETX") }()

	tmp := t.TempDir()
	installDir := filepath.Join(tmp, "krnr")
	_ = os.MkdirAll(installDir, 0o755)
	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	// Simulate system install by setting System=true and Path=installDir
	opts := Options{User: false, System: true, Path: installDir, From: src, DryRun: false, AddToPath: true}
	_, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall system add-to-path: %v", err)
	}
	m, err := loadMetadata()
	if err != nil {
		t.Fatalf("expected metadata for system install: %v", err)
	}
	if m.PathFile != "MachineEnv" {
		t.Fatalf("expected MachineEnv PathFile for system install, got: %s", m.PathFile)
	}
	// Now uninstall (test mode will not actually change PATH)
	actions, err := Uninstall(false)
	if err != nil {
		t.Fatalf("Uninstall failed for system: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("expected uninstall actions for system")
	}
	// The installDir should have been removed
	if _, err := os.Stat(installDir); err == nil {
		t.Fatalf("expected install directory to be removed, but it still exists: %s", installDir)
	}
}

func TestSystemInstall_SavesMetadataEvenWithoutAddToPath(t *testing.T) {
	// Verify that system installs record metadata even when not asked to modify PATH
	tmp := t.TempDir()
	// isolate data dir
	t.Setenv("KRNR_HOME", tmp)
	installDir := filepath.Join(tmp, "krnr")
	_ = os.MkdirAll(installDir, 0o755)
	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: false, System: true, Path: installDir, From: src, DryRun: false, AddToPath: false}
	_, err := ExecuteInstall(opts)
	if err != nil {
		t.Fatalf("ExecuteInstall system no-add-path: %v", err)
	}
	m, err := loadMetadata()
	if err != nil {
		t.Fatalf("expected metadata after system install: %v", err)
	}
	if m.TargetPath == "" {
		t.Fatalf("expected metadata TargetPath set")
	}
	if m.AddedToPath {
		t.Fatalf("expected AddedToPath to be false for no-add install")
	}
}

func TestSystemInstallDirectoryNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows; skip on Windows")
	}
	// Create a directory and remove write permission to simulate a protected system dir
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, "krnr")
	if err := os.MkdirAll(installDir, 0o555); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := Options{User: false, System: true, Path: installDir, From: src, DryRun: false}
	_, err := ExecuteInstall(opts)
	if err == nil {
		t.Fatalf("expected permission error when installing to non-writable system dir")
	}
	if !strings.Contains(err.Error(), "not writable") && !strings.Contains(err.Error(), "permission") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDetectBothScopes(t *testing.T) {
	// Simulate both user and system installs and ensure Status detects both
	tmp := t.TempDir()
	userDir := filepath.Join(tmp, "krnr", "bin")
	sysDir := filepath.Join(tmp, "sys")
	_ = os.MkdirAll(userDir, 0o755)
	_ = os.MkdirAll(sysDir, 0o755)
	// write dummy binaries
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	userBin := filepath.Join(userDir, binName)
	sysBin := filepath.Join(sysDir, binName)
	_ = os.WriteFile(userBin, []byte("u"), 0o644)
	_ = os.WriteFile(sysBin, []byte("s"), 0o644)
	// Set overrides so DefaultUserBin and systemBin point to these test dirs
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", filepath.Join(tmp))
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	if runtime.GOOS == "windows" {
		oldUser := os.Getenv("USERPROFILE")
		_ = os.Setenv("USERPROFILE", filepath.Join(tmp))
		defer func() { _ = os.Setenv("USERPROFILE", oldUser) }()
	}
	// override system bin via env var
	oldSys := os.Getenv("KRNR_TEST_SYSTEM_BIN")
	_ = os.Setenv("KRNR_TEST_SYSTEM_BIN", sysDir)
	defer func() { _ = os.Setenv("KRNR_TEST_SYSTEM_BIN", oldSys) }()
	// Ensure user bin is on PATH for process-level detection
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()
	_ = os.Setenv("PATH", oldPath+string(os.PathListSeparator)+filepath.Dir(userBin))
	// Now check Status
	st, err := GetStatus()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if !st.UserInstalled || !st.SystemInstalled {
		t.Fatalf("expected both user and system installs to be detected: %+v", st)
	}
	if !st.UserOnPath {
		t.Fatalf("expected user to be detected on PATH: %+v", st)
	}
}

func TestUninstallBothScopes(t *testing.T) {
	// Similar setup to ensure uninstall removes both scopes
	tmp := t.TempDir()
	userDir := filepath.Join(tmp, "krnr", "bin")
	sysDir := filepath.Join(tmp, "sys")
	_ = os.MkdirAll(userDir, 0o755)
	_ = os.MkdirAll(sysDir, 0o755)
	// write dummy binaries
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	userBin := filepath.Join(userDir, binName)
	sysBin := filepath.Join(sysDir, binName)
	_ = os.WriteFile(userBin, []byte("u"), 0o644)
	_ = os.WriteFile(sysBin, []byte("s"), 0o644)
	// Set overrides so DefaultUserBin and systemBin point to these test dirs
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", filepath.Join(tmp))
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	if runtime.GOOS == "windows" {
		oldUser := os.Getenv("USERPROFILE")
		_ = os.Setenv("USERPROFILE", filepath.Join(tmp))
		defer func() { _ = os.Setenv("USERPROFILE", oldUser) }()
	}
	// override system bin via env var
	oldSys := os.Getenv("KRNR_TEST_SYSTEM_BIN")
	_ = os.Setenv("KRNR_TEST_SYSTEM_BIN", sysDir)
	defer func() { _ = os.Setenv("KRNR_TEST_SYSTEM_BIN", oldSys) }()
	// Now call Uninstall; create metadata that points to user install
	_ = saveMetadata(filepath.Join(userDir, binName), true, "UserEnv", "")
	actions, err := Uninstall(false)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	// both files should be removed
	if _, err := os.Stat(filepath.Join(userDir, binName)); err == nil {
		t.Fatalf("expected user binary removed")
	}
	if _, err := os.Stat(filepath.Join(sysDir, binName)); err == nil {
		t.Fatalf("expected system binary removed")
	}
	if len(actions) == 0 {
		t.Fatalf("expected uninstall actions")
	}
}

func TestComputeNewPathString_RemovesOnlyTargetAndKeepsOthers(t *testing.T) {
	// simulate a PATH where one entry has doubled backslashes (corruption) and another is target
	cur := `C:\\Users\\Vox\\AppData\\Local\\Microsoft\\WindowsApps;C:\Temp`
	newPath, removed := computeNewPathString(cur, `C:\Temp`)
	if !removed {
		t.Fatalf("expected removal")
	}
	// ensure no doubled backslashes remain
	if strings.Contains(newPath, `\\\\`) {
		t.Fatalf("expected no doubled backslashes in resulting PATH, got: %s", newPath)
	}
	if strings.Contains(newPath, `C:\\Temp`) {
		t.Fatalf("expected C:\\Temp removed, got: %s", newPath)
	}
	if !strings.Contains(newPath, "WindowsApps") {
		t.Fatalf("expected WindowsApps to remain, got: %s", newPath)
	}
}
