package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// PlanUninstall returns a list of actions that would be performed by Uninstall.
func planWhenNoMetadata() []string {
	userTarget := filepath.Join(DefaultUserBin(), "krnr")
	if runtime.GOOS == "windows" {
		userTarget = userTarget + ".exe"
	}
	sysTarget := filepath.Join(systemBin(), "krnr")
	if runtime.GOOS == "windows" {
		sysTarget = sysTarget + ".exe"
	}
	return []string{
		"No install metadata found.",
		fmt.Sprintf("Check for binaries at: %s", userTarget),
		fmt.Sprintf("Or at system location: %s", sysTarget),
		"If you installed manually, remove those files and any PATH entries yourself, or re-run install with --add-to-path and then uninstall.",
		"If you encounter 'access denied' when uninstalling, run the downloaded krnr binary from a different directory (for example, your Downloads folder) so the installed binary isn't in use, then re-run uninstall.",
	}
}

func planWithMetadata(m *metadata) []string {
	actions := []string{fmt.Sprintf("Remove binary: %s", m.TargetPath)}
	// Also detect and mention user/system install targets if present
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	userTarget := filepath.Join(DefaultUserBin(), binName)
	sysTarget := filepath.Join(systemBin(), binName)
	if userTarget != m.TargetPath {
		addIfExists(userTarget, "Also remove user binary: %s", &actions)
	}
	if sysTarget != m.TargetPath {
		addIfExists(sysTarget, "Also remove system binary: %s", &actions)
	}
	if m.AddedToPath {
		if runtime.GOOS == "windows" {
			if m.PathFile == "MachineEnv" {
				actions = append(actions, "Remove PATH entries from Machine PATH (requires admin)")
			} else if m.OldUserPath != "" {
				actions = append(actions, "Restore user PATH to previous value")
			} else {
				actions = append(actions, "Remove PATH entry from user PATH (manual step)")
			}
		} else {
			rcfile := m.PathFile
			if rcfile == "" {
				rcfile = "~/.bashrc or ~/.profile"
			}
			actions = append(actions, fmt.Sprintf("Remove PATH entries from %s", rcfile))
		}
	}
	// Also suggest removing PATH entries for the other default locations
	if runtime.GOOS == "windows" {
		actions = append(actions, fmt.Sprintf("Check and remove %s from Machine and User PATH if present", filepath.Dir(sysTarget)))
		actions = append(actions, fmt.Sprintf("Check and remove %s from User PATH if present", filepath.Dir(userTarget)))
	}
	return actions

}

func addIfExists(path, fmtStr string, actions *[]string) {
	if _, err := os.Stat(path); err == nil {
		*actions = append(*actions, fmt.Sprintf(fmtStr, path))
	}
}

// PlanUninstall returns a list of actions that would be performed by Uninstall.
func PlanUninstall() ([]string, error) {
	m, err := loadMetadata()
	if err != nil {
		return planWhenNoMetadata(), nil
	}
	return planWithMetadata(m), nil
}
