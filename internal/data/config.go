package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds optional sync settings. Stored unencrypted at ~/.config/krypt/config.json.
type Config struct {
	SyncEnabled bool   `json:"sync_enabled"`
	GistID      string `json:"gist_id,omitempty"`
	Token       string `json:"token,omitempty"` // fallback; prefer KRYPT_GITHUB_TOKEN env
}

func cfgPath() (string, error) {
	base, err := configPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, cfgFile), nil
}

// LoadConfig reads the config file; returns default config if missing.
func LoadConfig() (*Config, error) {
	p, err := cfgPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes the config file.
func SaveConfig(cfg *Config) error {
	p, err := cfgPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(p, raw, 0600)
}
