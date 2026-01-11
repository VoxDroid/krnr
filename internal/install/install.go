// Package install provides installation utilities.
package install

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/VoxDroid/krnr/internal/config"
)

// Options controls install behavior.
type Options struct {
	User      bool
	System    bool
	Path      string
	From      string
	DryRun    bool
	Check     bool
	Yes       bool
	AddToPath bool // if true, installer will add target dir to PATH (with confirmation)
}

// DefaultUserBin returns a sensible per-user bin directory per OS.
func DefaultUserBin() string {
	// Use a per-application directory under the user's home for clarity and isolation.
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "krnr", "bin")
}

// systemBin returns a default system-wide bin directory for the OS.
func systemBin() string {
	if v := os.Getenv("KRNR_TEST_SYSTEM_BIN"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return `C:\\Program Files\\krnr`
	}
	return "/usr/local/bin"
}

// PlanInstall returns a list of human-readable actions that would be performed.
func PlanInstall(opts Options) ([]string, string, error) {
	// Resolve source
	src := opts.From
	if src == "" {
		ex, err := os.Executable()
		if err != nil {
			return nil, "", fmt.Errorf("determine current executable: %w", err)
		}
		src = ex
	}
	// Determine target dir
	targetDir := opts.Path
	if targetDir == "" {
		if opts.System {
			targetDir = systemBin()
		} else {
			targetDir = DefaultUserBin()
		}
	}
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	targetPath := filepath.Join(targetDir, binName)

	actions := []string{}
	actions = append(actions, fmt.Sprintf("Ensure directory exists: %s", targetDir))
	if src == targetPath {
		actions = append(actions, "No-op: source and destination are identical")
		return actions, targetPath, nil
	}
	actions = append(actions, fmt.Sprintf("Copy %s -> %s", src, targetPath))
	if runtime.GOOS != "windows" {
		actions = append(actions, fmt.Sprintf("Set executable bit on %s", targetPath))
	}
	// PATH hint
	pathEnv := os.Getenv("PATH")
	appendPathHints(&actions, pathEnv, targetDir)
	return actions, targetPath, nil

}

// appendPathHints appends human-friendly PATH hints to actions if the target directory
// is not currently on PATH.
func appendPathHints(actions *[]string, pathEnv, targetDir string) {
	if !ContainsPath(pathEnv, targetDir) {
		if runtime.GOOS == "windows" {
			*actions = append(*actions, fmt.Sprintf("Add %s to user PATH (e.g., via setx)", targetDir))
			*actions = append(*actions, fmt.Sprintf("(Suggestion) Add %s to your user PATH (run: setx PATH \"%%PATH%%;%s\")", targetDir, targetDir))
		} else {
			*actions = append(*actions, fmt.Sprintf("Add 'export PATH=\"%s:$PATH\"' to your shell rc (e.g., ~/.bashrc) or move to a location already on PATH", targetDir))
			*actions = append(*actions, fmt.Sprintf("(Suggestion) Add 'export PATH=\"%s:$PATH\"' to your shell rc (e.g., ~/.bashrc), or move binary into a directory already on PATH", targetDir))
		}
	}
}

// ContainsPath checks if the given directory is in the PATH environment variable.
func ContainsPath(pathEnv, dir string) bool {
	if pathEnv == "" || dir == "" {
		return false
	}
	// Normalize target
	dirClean := filepath.Clean(os.ExpandEnv(strings.TrimSpace(dir)))
	for _, p := range filepath.SplitList(pathEnv) {
		pClean := filepath.Clean(os.ExpandEnv(strings.TrimSpace(p)))
		if runtime.GOOS == "windows" {
			if strings.EqualFold(pClean, dirClean) {
				return true
			}
		} else {
			if pClean == dirClean {
				return true
			}
		}
	}
	return false
}

// metadata stores install operations to enable uninstall/rollback
type metadata struct {
	TargetPath  string    `json:"target_path"`
	AddedToPath bool      `json:"added_to_path"`
	PathFile    string    `json:"path_file,omitempty"`
	PathLine    string    `json:"path_line,omitempty"`
	OldUserPath string    `json:"old_user_path,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
}

func metadataPath() (string, error) {
	d, err := config.EnsureDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "install_metadata.json"), nil
}

func saveMetadata(target string, added bool, pathFile string, oldUserPath string) error {
	p, err := metadataPath()
	if err != nil {
		return err
	}
	m := metadata{TargetPath: target, AddedToPath: added, PathFile: pathFile, OldUserPath: oldUserPath, InstalledAt: time.Now()}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(p, b, 0o600)
}

func loadMetadata() (*metadata, error) {
	p, err := metadataPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var m metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func removeMetadata() error {
	p, err := metadataPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ExecuteInstall performs the install actions (will create dirs, copy file, set modes).
func ExecuteInstall(opts Options) ([]string, error) {
	actions, targetPath, err := PlanInstall(opts)
	if err != nil {
		return nil, err
	}
	if opts.DryRun {
		return actions, nil
	}
	// Perform actions
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, fmt.Errorf("create target dir: %w", err)
	}
	// resolve source executable
	src, err := resolveSourceExecutable(opts.From)
	if err != nil {
		return nil, fmt.Errorf("determine source executable: %w", err)
	}
	// Copy file
	if err := copyExecutable(src, targetPath); err != nil {
		return nil, err
	}

	// If requested, add to PATH (user or system mode)
	if opts.AddToPath {
		pathFile, oldPath, err := addToPath(targetPath, opts.System)
		if err != nil {
			return nil, fmt.Errorf("add to PATH: %w", err)
		}
		if err := saveMetadata(targetPath, true, pathFile, oldPath); err != nil {
			return nil, fmt.Errorf("save metadata: %w", err)
		}
	}

	return actions, nil
}

// resolveSourceExecutable resolves the source path for the install. If from is empty
// it returns the current running executable path.
func resolveSourceExecutable(from string) (string, error) {
	if from != "" {
		// If explicit source provided, ensure it exists
		if _, err := os.Stat(from); err != nil {
			return "", err
		}
		return from, nil
	}
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return ex, nil
}

// copyExecutable copies the source file to the destination atomically and sets
// executable permissions on non-Windows platforms.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = in.Close() }()
	// Ensure destination dir exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	tmpFile, terr := os.CreateTemp("", "krnr_tmp_")
	if terr != nil {
		return fmt.Errorf("create temp dest: %w", terr)
	}
	tmp := tmpFile.Name()
	// ensure temp file gets removed if something goes wrong
	defer func() { _ = os.Remove(tmp) }()
	// Write to temp file
	if _, err := io.Copy(tmpFile, in); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	// On non-windows ensure the executable bit is set
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmp, 0o755); err != nil {
			return fmt.Errorf("set exec bit: %w", err)
		}
	}
	if err := doRenameOrFallback(tmp, dst); err != nil {
		return err
	}
	return nil
}

// doRenameOrFallback attempts an atomic rename of tmp -> dst and falls back to a copy
// if the rename fails (useful on Windows where rename may fail if the target is in use).
func doRenameOrFallback(tmp, dst string) error {
	if err := os.Rename(tmp, dst); err == nil {
		return nil
	}
	renameErr := fmt.Errorf("rename failed")
	f, ferr := os.Open(tmp)
	if ferr != nil {
		return fmt.Errorf("rename: %v; fallback open tmp failed: %w", renameErr, ferr)
	}
	defer func() { _ = f.Close() }()
	dstF, derr := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if derr != nil {
		return fmt.Errorf("rename: %v; fallback open dst failed: %w", renameErr, derr)
	}
	_, copyErr := io.Copy(dstF, f)
	_ = dstF.Close()
	// Ensure tmp is cleaned up even if the OS has transient locks (Windows).
	for i := 0; i < 5; i++ {
		if rerr := os.Remove(tmp); rerr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if copyErr != nil {
		return fmt.Errorf("rename: %v; fallback copy failed: %w", renameErr, copyErr)
	}
	return nil
}

// Status represents the presence of krnr in user and system locations and PATH.
type Status struct {
	UserPath        string
	SystemPath      string
	UserInstalled   bool
	SystemInstalled bool
	UserOnPath      bool
	SystemOnPath    bool
	MetadataFound   bool
}

// GetStatus inspects the system and returns installation status for user and system locations.
func GetStatus() (*Status, error) {
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	userPath := filepath.Join(DefaultUserBin(), binName)
	sysPath := filepath.Join(systemBin(), binName)
	st := &Status{UserPath: userPath, SystemPath: sysPath}
	if _, err := os.Stat(userPath); err == nil {
		st.UserInstalled = true
	}
	if _, err := os.Stat(sysPath); err == nil {
		st.SystemInstalled = true
	}
	// Check PATH membership (current process PATH)
	st.UserOnPath = ContainsPath(os.Getenv("PATH"), filepath.Dir(userPath))
	if runtime.GOOS == "windows" {
		checkWindowsPathMembership(st)
	} else {
		st.SystemOnPath = ContainsPath(os.Getenv("PATH"), filepath.Dir(sysPath))
	}
	// metadata presence
	if p, err := metadataPath(); err == nil {
		if _, err := os.Stat(p); err == nil {
			st.MetadataFound = true
		}
	}
	// If PATH checks above didn't mark on-path, try resolving the command via LookPath/where
	adjustFromLookPath(st)
	return st, nil
}

// checkWindowsPathMembership inspects Machine and User PATH environment variables
// and updates the Status accordingly.
func checkWindowsPathMembership(st *Status) {
	// Check Machine and User PATH for system / user installs
	getCmd := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetEnvironmentVariable('Path','Machine')")
	if out, err := getCmd.Output(); err == nil {
		if ContainsPath(strings.TrimSpace(string(out)), filepath.Dir(st.SystemPath)) {
			st.SystemOnPath = true
		}
	}
	getCmd2 := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetEnvironmentVariable('Path','User')")
	if out2, err := getCmd2.Output(); err == nil {
		if ContainsPath(strings.TrimSpace(string(out2)), filepath.Dir(st.UserPath)) {
			st.UserOnPath = true
		}
	}
}

// adjustFromLookPath resolves the krnr executable via exec.LookPath and updates
// installation/path flags based on the resolved path.
func adjustFromLookPath(st *Status) {
	bin := "krnr"
	if runtime.GOOS == "windows" {
		bin = "krnr.exe"
	}
	if lp, err := exec.LookPath(bin); err == nil {
		lpClean := filepath.Clean(lp)
		if runtime.GOOS == "windows" {
			if strings.EqualFold(lpClean, filepath.Clean(st.UserPath)) {
				st.UserOnPath = true
				st.UserInstalled = true
			}
			if strings.EqualFold(lpClean, filepath.Clean(st.SystemPath)) {
				st.SystemOnPath = true
				st.SystemInstalled = true
			}
		} else {
			if lpClean == filepath.Clean(st.UserPath) {
				st.UserOnPath = true
				st.UserInstalled = true
			}
			if lpClean == filepath.Clean(st.SystemPath) {
				st.SystemOnPath = true
				st.SystemInstalled = true
			}
		}
	}
}

func addToPath(targetPath string, system bool) (string, string, error) {
	dir := filepath.Dir(targetPath)
	if runtime.GOOS == "windows" {
		return addToPathWindows(dir, system)
	}
	// Non-Windows behavior
	if system {
		// For system installs on Unix we prefer instructing the admin rather than editing system files.
		return "", "", fmt.Errorf("system PATH modifications require admin privileges; move the binary to %s or add %s to system PATH manually", systemBin(), filepath.Dir(targetPath))
	}
	return addToPathUnix(dir)
}

// addToPathWindows adds dir to the PATH for the given scope (User or Machine) and returns the path label and previous value
func addToPathWindows(dir string, system bool) (string, string, error) {
	// Decide scope: User or Machine
	scope := "User"
	pathLabel := "UserEnv"
	if system {
		scope = "Machine"
		pathLabel = "MachineEnv"
	}
	// Retrieve current PATH for scope
	getCmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("[Environment]::GetEnvironmentVariable('Path','%s')", scope))
	out, err := getCmd.Output()
	old := ""
	if err == nil {
		old = strings.TrimSpace(string(out))
	}
	// If already present in that PATH or current process PATH, nothing to do
	if ContainsPath(old, dir) || ContainsPath(os.Getenv("PATH"), dir) {
		return pathLabel, old, nil
	}
	// Test mode: don't modify actual environment
	if os.Getenv("KRNR_TEST_NO_SETX") != "" {
		return pathLabel, old, nil
	}
	// Compose new PATH and set it persistently
	newPath := old
	if newPath == "" {
		newPath = dir
	} else {
		newPath = newPath + ";" + dir
	}
	// Build an encoded PowerShell command to set the PATH safely (avoids quoting/escaping issues)
	script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, '%s')", toPowerShellString(newPath), scope)
	enc := encodePowerShellCommand(script)
	setCmd := exec.Command("powershell", "-NoProfile", "-EncodedCommand", enc)
	if out, err := setCmd.CombinedOutput(); err != nil {
		return "", old, fmt.Errorf("set %s PATH: %v (%s)", scope, err, string(out))
	}
	// Attempt to correct doubled backslashes if they appear after setting
	if fixMsg, fixed, err := ensureNoDoubleBackslashes(scope, false); err != nil {
		return "", old, fmt.Errorf("set %s PATH succeeded but post-fix failed: %v", scope, err)
	} else if fixed {
		_ = fixMsg // caller can log if needed; we return pathLabel
	}
	return pathLabel, old, nil
}

// addToPathUnix appends a PATH export to the appropriate shell rc file and returns the file path
func addToPathUnix(dir string) (string, string, error) {
	shell := os.Getenv("SHELL")
	homedir := os.Getenv("HOME")
	if homedir == "" {
		var err error
		homedir, err = os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
	}
	rcfile := filepath.Join(homedir, ".profile")
	// Prefer existing rc files if present (ensures tests that create .bashrc
	// are handled correctly regardless of the SHELL environment variable).
	if _, err := os.Stat(filepath.Join(homedir, ".bashrc")); err == nil {
		rcfile = filepath.Join(homedir, ".bashrc")
	} else if _, err := os.Stat(filepath.Join(homedir, ".zshrc")); err == nil {
		rcfile = filepath.Join(homedir, ".zshrc")
	} else if strings.Contains(shell, "zsh") {
		rcfile = filepath.Join(homedir, ".zshrc")
	} else if strings.Contains(shell, "bash") {
		rcfile = filepath.Join(homedir, ".bashrc")
	}
	line := fmt.Sprintf("# krnr: add %s to PATH\nexport PATH=\"%s:$PATH\"\n", dir, dir)
	f, err := os.OpenFile(rcfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line); err != nil {
		return "", "", err
	}
	_ = f.Sync()
	return rcfile, "", nil
}

// Uninstall removes previously installed binary and reverses PATH modifications.
func Uninstall(verbose bool) ([]string, error) {
	m, err := loadMetadata()
	if err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}
	actions := []string{}

	// 1) remove the installed target binary
	actions = append(actions, removeTargetBinary(m)...) // returns []string

	// 2) remove recorded PATH modification
	if m.AddedToPath {
		if runtime.GOOS == "windows" {
			if msgs, err := removeWindowsPathEntry(m, verbose); err != nil {
				actions = append(actions, fmt.Sprintf("failed to remove PATH entry (windows): %v", err))
			} else {
				actions = append(actions, msgs...)
			}
		} else {
			msgs := removeUnixPathEntry(m)
			actions = append(actions, msgs...)
		}
	}

	// 3) remove other installations if present
	actions = append(actions, removeOtherInstallations(m)...) // returns []string

	// 4) ensure PATH entries removed for both scopes (windows helper)
	if runtime.GOOS == "windows" {
		if msg, err := removeFromPathWindows("User", filepath.Dir(filepath.Join(DefaultUserBin(), "krnr.exe")), verbose); err == nil && msg != "" {
			actions = append(actions, msg)
		}
		if msg, err := removeFromPathWindows("Machine", filepath.Dir(filepath.Join(systemBin(), "krnr.exe")), verbose); err == nil && msg != "" {
			actions = append(actions, msg)
		}
	}

	_ = removeMetadata()
	return actions, nil
}

// Helper: remove the recorded target binary and return human-readable actions
func removeTargetBinary(m *metadata) []string {
	actions := []string{}
	if _, err := os.Stat(m.TargetPath); err == nil {
		if err := os.Remove(m.TargetPath); err != nil {
			actions = append(actions, fmt.Sprintf("Failed to remove %s: %v", m.TargetPath, err))
			actions = append(actions, "If the file is in use, run the downloaded krnr binary outside the installation directory (for example from Downloads) and re-run uninstall, or stop any running instances and retry.")
		} else {
			actions = append(actions, fmt.Sprintf("Removed %s", m.TargetPath))
			// Attempt to remove parent directory if empty
			parent := filepath.Dir(m.TargetPath)
			if err := os.Remove(parent); err == nil {
				actions = append(actions, fmt.Sprintf("Removed empty directory %s", parent))
			} else if pe, ok := err.(*os.PathError); ok {
				// If it's not removed because not empty, report contents
				if !os.IsNotExist(pe.Err) {
					if entries, re := os.ReadDir(parent); re == nil {
						names := []string{}
						for _, e := range entries {
							names = append(names, e.Name())
						}
						if len(names) > 0 {
							actions = append(actions, fmt.Sprintf("Directory %s not empty; contains: %s - please remove manually if desired", parent, strings.Join(names, ", ")))
						}
					}
				}
			}
		}
	} else {
		actions = append(actions, fmt.Sprintf("Target %s not found; skipping", m.TargetPath))
	}
	return actions
}

// Helper: remove PATH entry on Unix-like systems, if present
func removeUnixPathEntry(m *metadata) []string {
	actions := []string{}
	rcfile := m.PathFile
	if rcfile == "" {
		homedir, _ := os.UserHomeDir()
		rcfile = filepath.Join(homedir, ".profile")
		if _, err := os.Stat(filepath.Join(homedir, ".bashrc")); err == nil {
			rcfile = filepath.Join(homedir, ".bashrc")
		}
	}
	b, err := os.ReadFile(rcfile)
	if err == nil {
		s := string(b)
		newLines := []string{}
		for _, line := range strings.Split(s, "\n") {
			if strings.Contains(line, filepath.Dir(m.TargetPath)) && strings.Contains(line, "krnr") {
				continue
			}
			newLines = append(newLines, line)
		}
		_ = os.WriteFile(rcfile, []byte(strings.Join(newLines, "\n")), 0o644)
		actions = append(actions, fmt.Sprintf("Removed PATH entry from %s", rcfile))
	} else {
		actions = append(actions, fmt.Sprintf("No PATH file %s found; nothing to remove", rcfile))
	}
	return actions
}

// Helper: remove recorded PATH entry on Windows (restores or removes depending on metadata)
func removeWindowsPathEntry(m *metadata, verbose bool) ([]string, error) {
	actions := []string{}
	scope := "User"
	if m.PathFile == "MachineEnv" {
		scope = "Machine"
	}
	d := filepath.Dir(m.TargetPath)
	if m.OldUserPath != "" {
		if os.Getenv("KRNR_TEST_NO_SETX") != "" {
			actions = append(actions, fmt.Sprintf("Note: would restore %s PATH to previous value (test mode)", scope))
			return actions, nil
		}
		// Attempt full restore
		script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, '%s')", toPowerShellString(m.OldUserPath), scope)
		enc := encodePowerShellCommand(script)
		setCmd := exec.Command("powershell", "-NoProfile", "-EncodedCommand", enc)
		if out, err := setCmd.CombinedOutput(); err != nil {
			actions = append(actions, fmt.Sprintf("failed to restore %s PATH: %v (%s)", scope, err, string(out)))
			return actions, nil
		}
		actions = append(actions, fmt.Sprintf("Restored %s PATH to previous value", scope))
		if fixMsg, fixed, err := ensureNoDoubleBackslashes(scope, verbose); err != nil {
			actions = append(actions, fmt.Sprintf("set %s PATH succeeded but post-fix failed: %v", scope, err))
		} else if fixed {
			actions = append(actions, fixMsg)
		}
		return actions, nil
	}
	// Otherwise remove specific entry
	msg, err := removeFromPathWindows(scope, d, verbose)
	if err != nil {
		return actions, err
	}
	actions = append(actions, msg)
	return actions, nil
}

// Helper: remove other installations (user/system targets) that are not the recorded target
func removeOtherInstallations(m *metadata) []string {
	actions := []string{}
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	userTarget := filepath.Join(DefaultUserBin(), binName)
	sysTarget := filepath.Join(systemBin(), binName)
	if userTarget != m.TargetPath {
		if _, err := os.Stat(userTarget); err == nil {
			if err := os.Remove(userTarget); err != nil {
				actions = append(actions, fmt.Sprintf("Failed to remove %s: %v", userTarget, err))
			} else {
				actions = append(actions, fmt.Sprintf("Removed %s", userTarget))
				_ = os.Remove(filepath.Dir(userTarget))
			}
		}
	}
	if sysTarget != m.TargetPath {
		if _, err := os.Stat(sysTarget); err == nil {
			if err := os.Remove(sysTarget); err != nil {
				actions = append(actions, fmt.Sprintf("Failed to remove %s: %v", sysTarget, err))
			} else {
				actions = append(actions, fmt.Sprintf("Removed %s", sysTarget))
				_ = os.Remove(filepath.Dir(sysTarget))
			}
		}
	}
	return actions
}

// removeFromPathWindows removes a single directory from the PATH for the given scope (User or Machine).
// Returns a human-readable action message and an error if the operation failed.
// computeNewPathString computes the new PATH value when removing dir from cur PATH string.
// It returns the new PATH and a boolean indicating whether an entry was removed.
func normalizeBackslashes(s string) string {
	// Collapse repeated backslashes into a single backslash
	for strings.Contains(s, "\\\\") {
		s = strings.ReplaceAll(s, "\\\\", "\\")
	}
	return s
}

func ensureNoDoubleBackslashes(scope string, verbose bool) (string, bool, error) {
	// read current PATH
	getCmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("[Environment]::GetEnvironmentVariable('Path','%s')", scope))
	out, err := getCmd.Output()
	cur := ""
	if err == nil {
		cur = string(out)
	}
	if strings.Contains(cur, "\\\\") {
		fixed := cur
		for strings.Contains(fixed, "\\\\") {
			fixed = strings.ReplaceAll(fixed, "\\\\", "\\")
		}
		// set fixed value
		script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, '%s')", toPowerShellString(fixed), scope)
		enc := encodePowerShellCommand(script)
		setCmd := exec.Command("powershell", "-NoProfile", "-EncodedCommand", enc)
		if out2, err := setCmd.CombinedOutput(); err != nil {
			return "", false, fmt.Errorf("failed to fix doubled backslashes in %s PATH: %v (%s)", scope, err, string(out2))
		}
		if verbose {
			return fmt.Sprintf("Fixed doubled backslashes in %s PATH. Before: %q. After: %q", scope, cur, fixed), true, nil
		}
		return "Fixed doubled backslashes in PATH", true, nil
	}
	return "", false, nil
}

func computeNewPathString(cur, dir string) (string, bool) {
	parts := strings.Split(cur, ";")
	newParts := []string{}
	removed := false
	for _, p := range parts {
		pTrim := strings.TrimSpace(p)
		if pTrim == "" {
			continue
		}
		pClean := filepath.Clean(pTrim)
		if runtime.GOOS == "windows" {
			pClean = normalizeBackslashes(pClean)
			if strings.EqualFold(pClean, normalizeBackslashes(filepath.Clean(dir))) {
				removed = true
				continue
			}
		} else {
			if pClean == filepath.Clean(dir) {
				removed = true
				continue
			}
		}
		newParts = append(newParts, pClean)
	}
	return strings.Join(newParts, ";"), removed
}

func toPowerShellString(s string) string {
	// Return a single-quoted PowerShell string literal safely (escape single quotes by doubling them)
	escaped := strings.ReplaceAll(s, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

func encodePowerShellCommand(s string) string {
	// PowerShell -EncodedCommand expects base64-encoded UTF-16LE
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[2*i:], v)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func removeFromPathWindows(scope, dir string, verbose bool) (string, error) {
	// Read current PATH for scope
	getCmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("[Environment]::GetEnvironmentVariable('Path','%s')", scope))
	out, err := getCmd.Output()
	cur := ""
	if err == nil {
		cur = strings.TrimSpace(string(out))
	}
	if cur == "" {
		return fmt.Sprintf("No %s PATH found; nothing to remove", scope), nil
	}
	newPath, removed := computeNewPathString(cur, dir)
	if !removed {
		return fmt.Sprintf("No %s PATH entry for %s found; nothing to do", scope, dir), nil
	}
	if os.Getenv("KRNR_TEST_NO_SETX") != "" {
		return fmt.Sprintf("Note: would remove %s from %s PATH (test mode)", dir, scope), nil
	}
	// Build an encoded PowerShell command to set the PATH safely
	script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, '%s')", toPowerShellString(newPath), scope)
	enc := encodePowerShellCommand(script)
	setCmd := exec.Command("powershell", "-NoProfile", "-EncodedCommand", enc)
	if out2, err := setCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to set %s PATH: %v (%s)", scope, err, string(out2))
	}
	// Attempt to correct doubled backslashes if they appear after setting
	if fixMsg, fixed, err := ensureNoDoubleBackslashes(scope, verbose); err != nil {
		return "", fmt.Errorf("set %s PATH succeeded but post-fix failed: %v", scope, err)
	} else if fixed {
		if verbose {
			return fmt.Sprintf("Removed %s from %s PATH; %s", dir, scope, fixMsg), nil
		}
		return fmt.Sprintf("Removed %s from %s PATH", dir, scope), nil
	}
	return fmt.Sprintf("Removed %s from %s PATH", dir, scope), nil
}
