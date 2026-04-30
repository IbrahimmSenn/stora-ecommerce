package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	e, err := NewEncryptor(hex.EncodeToString(raw))
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	return e
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	e := newTestEncryptor(t)
	cases := []string{"hello", "m.ibrahimsenn@gmail.com", "1600 Amphitheatre Parkway", strings.Repeat("x", 4096)}
	for _, pt := range cases {
		ct, err := e.Encrypt(pt)
		if err != nil {
			t.Fatalf("encrypt %q: %v", pt, err)
		}
		got, err := e.Decrypt(ct)
		if err != nil {
			t.Fatalf("decrypt %q: %v", pt, err)
		}
		if got != pt {
			t.Fatalf("roundtrip mismatch: got %q want %q", got, pt)
		}
	}
}

func TestEmptyStringRoundtrip(t *testing.T) {
	e := newTestEncryptor(t)
	ct, err := e.Encrypt("")
	if err != nil || ct != nil {
		t.Fatalf("empty encrypt: ct=%v err=%v", ct, err)
	}
	got, err := e.Decrypt(nil)
	if err != nil || got != "" {
		t.Fatalf("empty decrypt: got=%q err=%v", got, err)
	}
}

func TestNonceIsUnique(t *testing.T) {
	e := newTestEncryptor(t)
	a, _ := e.Encrypt("same plaintext")
	b, _ := e.Encrypt("same plaintext")
	if bytes.Equal(a, b) {
		t.Fatal("expected different ciphertexts for repeated plaintext (nonce reuse)")
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	e := newTestEncryptor(t)
	ct, _ := e.Encrypt("payload")
	ct[len(ct)-1] ^= 0x01
	if _, err := e.Decrypt(ct); err == nil {
		t.Fatal("expected decrypt to fail on tampered ciphertext")
	}
}

func TestWrongKeyFails(t *testing.T) {
	a := newTestEncryptor(t)
	b := newTestEncryptor(t)
	ct, _ := a.Encrypt("secret")
	if _, err := b.Decrypt(ct); err == nil {
		t.Fatal("expected decrypt with wrong key to fail")
	}
}

func TestBadKey(t *testing.T) {
	if _, err := NewEncryptor("not-hex"); err == nil {
		t.Fatal("expected error for non-hex key")
	}
	if _, err := NewEncryptor(hex.EncodeToString(make([]byte, 16))); err == nil {
		t.Fatal("expected error for 16-byte key")
	}
}
