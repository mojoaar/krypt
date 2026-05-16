package data

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const maxAttempts = 5

// attemptsFile is the on-disk format; MAC is HMAC-SHA256(secret, "krypt:attempts:N").
type attemptsFile struct {
	Attempts int    `json:"attempts"`
	MAC      string `json:"mac"`
}

// secretPath is where the per-install HMAC signing key lives (mode 0600).
func secretPath() string {
	return filepath.Join(attemptsDir(), ".vault-secret")
}

// loadOrCreateSecret returns the 32-byte signing key, generating it on first run.
func loadOrCreateSecret() ([]byte, error) {
	p := secretPath()
	b, err := os.ReadFile(p)
	if err == nil && len(b) == 32 {
		return b, nil
	}
	// Generate a new secret.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	dir := attemptsDir()
	if dir == "" {
		return nil, fmt.Errorf("cannot resolve config dir")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(p, secret, 0600); err != nil {
		return nil, fmt.Errorf("write secret: %w", err)
	}
	return secret, nil
}

// computeMAC returns hex(HMAC-SHA256(secret, "krypt:attempts:N")).
func computeMAC(secret []byte, n int) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("krypt:attempts:%d", n)))
	return hex.EncodeToString(mac.Sum(nil))
}

func attemptsDir() string {
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return ""
		}
		return filepath.Join(dir, configDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", configDir)
}

func attemptsPath() string {
	return filepath.Join(attemptsDir(), "attempts.json")
}

// LoadAttempts returns the current failed-attempt count.
// If the file is present but the MAC is invalid (tampered), it returns maxAttempts
// so the lockout cannot be bypassed by editing the file.
// A missing file is treated as 0 (normal first-run behaviour).
// A file with no MAC (old format) and 0 attempts is treated as a clean migration —
// the file is rewritten with a valid MAC. A no-MAC file with >0 attempts is treated
// as tampered to prevent a downgrade attack.
func LoadAttempts() int {
	b, err := os.ReadFile(attemptsPath())
	if err != nil {
		// File does not exist — fresh install, no failed attempts.
		return 0
	}
	var af attemptsFile
	if err := json.Unmarshal(b, &af); err != nil {
		// Unreadable → treat as tampered.
		return maxAttempts
	}
	// Migration: old format had no MAC field.
	if af.MAC == "" {
		if af.Attempts > 0 {
			// Cannot verify — treat as tampered to prevent downgrade attack.
			return maxAttempts
		}
		// attempts == 0 and no MAC: safe to migrate; rewrite with HMAC.
		saveAttempts(0)
		return 0
	}
	secret, err := loadOrCreateSecret()
	if err != nil {
		// Cannot load secret — treat as tampered to stay safe.
		return maxAttempts
	}
	expected := computeMAC(secret, af.Attempts)
	if !hmac.Equal([]byte(af.MAC), []byte(expected)) {
		// MAC mismatch → tampered.
		return maxAttempts
	}
	return af.Attempts
}

// IncrementAttempts records one more failed attempt and returns the new count.
func IncrementAttempts() int {
	n := LoadAttempts() + 1
	// Cap at maxAttempts so we don't exceed it on tamper-detection increments.
	if n > maxAttempts {
		n = maxAttempts
	}
	saveAttempts(n)
	return n
}

// ResetAttempts clears the failed-attempt counter after a successful unlock.
func ResetAttempts() {
	saveAttempts(0)
}

// MaxAttempts returns the configured maximum before vault destruction.
func MaxAttempts() int { return maxAttempts }

// DestroyVault deletes vault.enc and 2fa.enc, then resets the counter.
func DestroyVault() {
	if p, err := vaultPath(); err == nil {
		_ = os.Remove(p)
	}
	if p, err := twoFAPath(); err == nil {
		_ = os.Remove(p)
	}
	ResetAttempts()
}

func saveAttempts(n int) {
	secret, err := loadOrCreateSecret()
	if err != nil {
		return
	}
	dir := attemptsDir()
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0700)
	af := attemptsFile{
		Attempts: n,
		MAC:      computeMAC(secret, n),
	}
	b, _ := json.Marshal(af)
	_ = os.WriteFile(attemptsPath(), b, 0600)
}
