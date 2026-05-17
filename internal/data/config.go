package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds optional sync and generator settings.
// Stored unencrypted at ~/.config/krypt/config.json.
type Config struct {
	SyncEnabled bool              `json:"sync_enabled"`
	GistID      string            `json:"gist_id,omitempty"`
	Token       string            `json:"token,omitempty"` // fallback; prefer KRYPT_GITHUB_TOKEN env
	PasswordGen PasswordGenConfig `json:"password_gen,omitempty"`
}

// PasswordGenConfig controls the password generator. All fields are optional;
// zero values fall back to the built-in defaults.
type PasswordGenConfig struct {
	Length    int    `json:"length,omitempty"`    // default 30
	Uppercase bool   `json:"uppercase,omitempty"` // default true
	Lowercase bool   `json:"lowercase,omitempty"` // default true
	Digits    bool   `json:"digits,omitempty"`    // default true
	Symbols   bool   `json:"symbols,omitempty"`   // default true
	SymbolSet string `json:"symbol_set,omitempty"` // default "!@#$%^&*-_+=?"
}

// Charset returns the character set to use for password generation,
// applying defaults for any zero/false values.
func (p PasswordGenConfig) Charset() string {
	// All false means "use defaults" (not "use nothing")
	noneSet := !p.Uppercase && !p.Lowercase && !p.Digits && !p.Symbols
	upper := p.Uppercase || noneSet
	lower := p.Lowercase || noneSet
	digits := p.Digits || noneSet
	symbols := p.Symbols || noneSet

	cs := ""
	if lower {
		cs += "abcdefghijklmnopqrstuvwxyz"
	}
	if upper {
		cs += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if digits {
		cs += "0123456789"
	}
	if symbols {
		sym := p.SymbolSet
		if sym == "" {
			sym = "!@#$%^&*-_+=?"
		}
		cs += sym
	}
	return cs
}

// EffectiveLength returns the password length, defaulting to 30.
func (p PasswordGenConfig) EffectiveLength() int {
	if p.Length <= 0 {
		return 30
	}
	return p.Length
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
