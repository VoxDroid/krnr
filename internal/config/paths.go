package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Environment variables
const (
	EnvKRNRHome = "KRNR_HOME" // override data directory
	EnvKRNRDB   = "KRNR_DB"   // override DB file path
)

// DataDir returns the directory used to store krnr data.
// Precedence: KRNR_HOME env > default (~/.krnr)
func DataDir() (string, error) {
	if v := os.Getenv(EnvKRNRHome); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Use a dot-directory in the user's home on all platforms
	return filepath.Join(home, ".krnr"), nil
}

// EnsureDataDir makes sure the data directory exists and is writable.
func EnsureDataDir() (string, error) {
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// DBPath returns the full path to the SQLite database file.
// Precedence: KRNR_DB env > KRNR_HOME/env DataDir
func DBPath() (string, error) {
	if v := os.Getenv(EnvKRNRDB); v != "" {
		return v, nil
	}
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	if d == "" {
		return "", errors.New("data dir not available")
	}
	return filepath.Join(d, "krnr.db"), nil
}

// DefaultShellHint returns a platform-appropriate shell hint string for docs/help.
func DefaultShellHint() string {
	if runtime.GOOS == "windows" {
		return "cmd /C <cmd> or pwsh -Command <cmd>"
	}
	return "bash -c <cmd>"
}
