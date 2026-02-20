package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "diary-cli", "profile.json"), nil
}

func Load(path string) (*UserProfile, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return NewEmpty(), err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewEmpty(), nil
		}
		return NewEmpty(), fmt.Errorf("failed to read profile: %w", err)
	}

	var p UserProfile
	if err := json.Unmarshal(data, &p); err != nil {
		backupPath := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102-150405"))
		_ = os.WriteFile(backupPath, data, 0600)
		return NewEmpty(), fmt.Errorf("failed to parse profile (backup created: %s): %w", backupPath, err)
	}

	if p.Version == 0 {
		p.Version = CurrentVersion
	}

	return &p, nil
}

func Save(path string, p *UserProfile) error {
	if p == nil {
		p = NewEmpty()
	}
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	p.Version = CurrentVersion
	p.UpdatedAt = nowRFC3339()

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp profile: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to persist profile: %w", err)
	}

	return nil
}
