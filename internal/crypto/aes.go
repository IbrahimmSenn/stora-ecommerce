// Package crypto provides app-level AES-256-GCM encryption for PII stored at rest.
// Ciphertext format: nonce || gcm_ciphertext_with_tag.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type Encryptor struct {
	gcm cipher.AEAD
	key []byte // raw 32-byte key — held to derive HMAC lookup digests.
}

// NewEncryptor builds an AES-256-GCM encryptor. hexKey must decode to 32 bytes.
// Generate one with: openssl rand -hex 32
func NewEncryptor(hexKey string) (*Encryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("encryption key must be hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}
	return &Encryptor{gcm: gcm, key: key}, nil
}

// HMAC returns a deterministic 32-byte HMAC-SHA256 of v under the encryption
// key. Use as an opaque equality-lookup index for an encrypted column when
// SELECT-by-value is required (e.g. payments.stripe_payment_intent_id — the
// webhook arrives with the plaintext id and needs to find the row). Returns
// nil for empty input so callers can store SQL NULL.
func (e *Encryptor) HMAC(v string) []byte {
	if v == "" {
		return nil
	}
	h := hmac.New(sha256.New, e.key)
	h.Write([]byte(v))
	return h.Sum(nil)
}

// Encrypt returns (nil, nil) for empty input so callers can store SQL NULL
// for optional fields without a separate branch.
func (e *Encryptor) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}
	return e.gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt returns "" for nil/empty input (mirror of Encrypt's empty case).
func (e *Encryptor) Decrypt(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	ns := e.gcm.NonceSize()
	if len(ciphertext) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, payload := ciphertext[:ns], ciphertext[ns:]
	plain, err := e.gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}
