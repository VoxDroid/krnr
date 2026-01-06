package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv(EnvKRNRHome, tmp)
	defer os.Unsetenv(EnvKRNRHome)

	d, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir(): %v", err)
	}
	if d != tmp {
		t.Fatalf("expected %s got %s", tmp, d)
	}
}

func TestDBPathEnvOverride(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "custom.db")
	os.Setenv(EnvKRNRDB, tmp)
	defer os.Unsetenv(EnvKRNRDB)

	p, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath(): %v", err)
	}
	if p != tmp {
		t.Fatalf("expected %s got %s", tmp, p)
	}
}

func TestEnsureDataDirCreatesDir(t *testing.T) {
	os.Unsetenv(EnvKRNRHome)
	tmp := t.TempDir()
	// fake home by setting HOME/USERPROFILE
	os.Setenv("HOME", tmp)
	os.Setenv("USERPROFILE", tmp)

	d, err := EnsureDataDir()
	if err != nil {
		t.Fatalf("EnsureDataDir(): %v", err)
	}
	if _, err := os.Stat(d); err != nil {
		t.Fatalf("expected dir %s to exist: %v", d, err)
	}
}
