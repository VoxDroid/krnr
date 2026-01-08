package user

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/VoxDroid/krnr/internal/config"
)

// Profile holds persisted author metadata.
type Profile struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

func profilePath() (string, error) {
	d, err := config.EnsureDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "whoami.json"), nil
}

// SetProfile saves the author profile to disk.
func SetProfile(p Profile) error {
	pfile, err := profilePath()
	if err != nil {
		return err
	}
	f, err := os.Create(pfile)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

// GetProfile reads the author profile. Returns (Profile, true, nil) if found.
func GetProfile() (Profile, bool, error) {
	pfile, err := profilePath()
	if err != nil {
		return Profile{}, false, err
	}
	b, err := os.ReadFile(pfile)
	if err != nil {
		if os.IsNotExist(err) {
			return Profile{}, false, nil
		}
		return Profile{}, false, err
	}
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, false, err
	}
	return p, true, nil
}

// ClearProfile removes the persisted profile.
func ClearProfile() error {
	pfile, err := profilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(pfile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
