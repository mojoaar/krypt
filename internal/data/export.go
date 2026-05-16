package data

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExportJSON marshals entries to indented JSON and writes the file at path
// with owner-read-only permissions.
func ExportJSON(entries []Entry, path string) error {
	blob, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, blob, 0600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// ExportEncryptedJSON marshals entries to JSON then encrypts the result
// with Argon2id + AES-256-GCM using passphrase. The resulting binary file
// can be decrypted with DecryptExport.
func ExportEncryptedJSON(entries []Entry, passphrase, path string) error {
	plain, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	blob, err := encrypt(plain, passphrase)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	if err := os.WriteFile(path, blob, 0600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// DecryptExport decrypts a file produced by ExportEncryptedJSON and returns
// the plaintext JSON bytes.
func DecryptExport(path, passphrase string) ([]byte, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return decrypt(blob, passphrase)
}
