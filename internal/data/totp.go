package data

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // TOTP spec requires SHA-1
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// GenerateTOTPSecret creates a random 20-byte base32-encoded TOTP secret.
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "="), nil
}

// GenerateTOTP returns the current 6-digit TOTP code and the seconds remaining
// in the current 30-second window. Returns an error if the secret is invalid.
func GenerateTOTP(secret string) (code string, remaining int, err error) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	// Pad to a multiple of 8 for base32 decoding.
	if n := len(secret) % 8; n != 0 {
		secret += strings.Repeat("=", 8-n)
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", 0, fmt.Errorf("invalid TOTP secret: %w", err)
	}

	now := time.Now().Unix()
	counter := now / 30
	remaining = 30 - int(now%30)

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter)) //nolint:gosec

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	value := int64(h[offset]&0x7f)<<24 |
		int64(h[offset+1])<<16 |
		int64(h[offset+2])<<8 |
		int64(h[offset+3])

	otp := value % 1_000_000
	return fmt.Sprintf("%06d", otp), remaining, nil
}
