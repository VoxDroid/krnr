package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the directory used to store krnr data.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Use a dot-directory in the user's home on all platforms
	if runtime.GOOS == "windows" {
		return filepath.Join(home, ".krnr"), nil
	}
	return filepath.Join(home, ".krnr"), nil
}

// DBPath returns the full path to the SQLite database file.
func DBPath() (string, error) {
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "krnr.db"), nil
}
