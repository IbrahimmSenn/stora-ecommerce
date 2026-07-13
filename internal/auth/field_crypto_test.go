package auth

import (
	"testing"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
)

const testHexKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestAuthRepo(t *testing.T) *postgresAuthRepository {
	t.Helper()
	enc, err := crypto.NewEncryptor(testHexKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	return &postgresAuthRepository{enc: enc}
}

func TestFieldCrypto_RoundTrip(t *testing.T) {
	r := newTestAuthRepo(t)
	const secret = "JBSWY3DPEHPK3PXP" // a TOTP base32 secret

	ct, err := r.encField(secret)
	if err != nil {
		t.Fatalf("encField: %v", err)
	}
	if ct == secret {
		t.Fatal("stored value must not equal the plaintext secret")
	}

	pt, err := r.decField(ct)
	if err != nil {
		t.Fatalf("decField: %v", err)
	}
	if pt != secret {
		t.Fatalf("round-trip mismatch: got %q want %q", pt, secret)
	}
}

func TestFieldCrypto_EmptyPassesThrough(t *testing.T) {
	r := newTestAuthRepo(t)
	if v, err := r.encField(""); err != nil || v != "" {
		t.Fatalf("encField(empty) = %q, %v", v, err)
	}
	if v, err := r.decField(""); err != nil || v != "" {
		t.Fatalf("decField(empty) = %q, %v", v, err)
	}
}
