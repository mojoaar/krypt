package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/argon2"
)

const (
	configDir = "krypt"
	vaultFile = "vault.enc"
	twoFAFile = "2fa.enc"
	cfgFile   = "config.json"

	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32

	saltLen  = 16
	nonceLen = 12
)

// configPath returns ~/.config/krypt on macOS/Linux, %AppData%\krypt on Windows.
func configPath() (string, error) {
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("config dir: %w", err)
		}
		return filepath.Join(dir, configDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", configDir), nil
}

func vaultPath() (string, error) {
	base, err := configPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, vaultFile), nil
}

func twoFAPath() (string, error) {
	base, err := configPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, twoFAFile), nil
}

// deriveKey uses Argon2id to derive a 32-byte key from a password and salt.
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// encrypt encrypts plaintext with AES-256-GCM. Returns [salt(16)][nonce(12)][ciphertext].
func encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("rand salt: %w", err)
	}
	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// decrypt decrypts a blob produced by encrypt.
func decrypt(blob []byte, password string) ([]byte, error) {
	if len(blob) < saltLen+nonceLen+1 {
		return nil, fmt.Errorf("blob too short")
	}
	salt := blob[:saltLen]
	nonce := blob[saltLen : saltLen+nonceLen]
	ciphertext := blob[saltLen+nonceLen:]

	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: wrong password or corrupted vault")
	}
	return plain, nil
}

// Store holds the decrypted vault and its disk path.
type Store struct {
	data VaultData
	path string
}

// VaultExists reports whether a vault file exists on disk.
func VaultExists() (bool, error) {
	p, err := vaultPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Load decrypts and loads the vault. Returns an empty store if no vault exists yet.
func Load(masterPassword string) (*Store, error) {
	p, err := vaultPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	s := &Store{path: p, data: VaultData{Entries: []Entry{}}}

	blob, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read vault: %w", err)
	}

	plain, err := decrypt(blob, masterPassword)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(plain, &s.data); err != nil {
		return nil, fmt.Errorf("parse vault: %w", err)
	}
	return s, nil
}

// Save encrypts the vault and writes it to disk.
func (s *Store) Save(masterPassword string) error {
	plain, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	blob, err := encrypt(plain, masterPassword)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, blob, 0600)
}

// Path returns the vault file path.
func (s *Store) Path() string { return s.path }

// EncryptedBlob returns the AES-256-GCM encrypted vault bytes (for sync).
func (s *Store) EncryptedBlob(masterPassword string) ([]byte, error) {
	plain, err := json.Marshal(s.data)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return encrypt(plain, masterPassword)
}

// Entries returns a copy of all entries.
func (s *Store) Entries() []Entry { return s.data.Entries }

// Add adds an entry and returns it with its generated ID.
func (s *Store) Add(e Entry) Entry {
	e.ID = newID()
	if e.Tags == nil {
		e.Tags = []string{}
	}
	s.data.Entries = append(s.data.Entries, e)
	return e
}

// Update replaces the entry with matching ID.
func (s *Store) Update(e Entry) {
	for i, v := range s.data.Entries {
		if v.ID == e.ID {
			s.data.Entries[i] = e
			return
		}
	}
}

// Delete removes the entry with the given ID.
func (s *Store) Delete(id string) {
	filtered := s.data.Entries[:0]
	for _, v := range s.data.Entries {
		if v.ID != id {
			filtered = append(filtered, v)
		}
	}
	s.data.Entries = filtered
}

// TwoFAEnabled reports whether a 2FA secret file exists.
func TwoFAEnabled() (bool, error) {
	p, err := twoFAPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// LoadTwoFASecret decrypts and returns the TOTP secret for app-level 2FA.
func LoadTwoFASecret(masterPassword string) (string, error) {
	p, err := twoFAPath()
	if err != nil {
		return "", err
	}
	blob, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("read 2fa: %w", err)
	}
	plain, err := decrypt(blob, masterPassword)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// SaveTwoFASecret encrypts and stores the TOTP secret for app-level 2FA.
func SaveTwoFASecret(secret, masterPassword string) error {
	p, err := twoFAPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	blob, err := encrypt([]byte(secret), masterPassword)
	if err != nil {
		return err
	}
	return os.WriteFile(p, blob, 0600)
}

// DisableTwoFA deletes the 2FA secret file.
func DisableTwoFA() error {
	p, err := twoFAPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
