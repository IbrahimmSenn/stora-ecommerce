package tlsutil

import (
	"crypto/tls"
	"path/filepath"
	"testing"
)

func TestEnsureSelfSigned_GeneratesUsableKeypair(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "server.crt")
	key := filepath.Join(dir, "server.key")

	if err := EnsureSelfSigned(cert, key); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// The pair must load as a valid TLS certificate.
	if _, err := tls.LoadX509KeyPair(cert, key); err != nil {
		t.Fatalf("generated keypair does not load: %v", err)
	}
}

func TestEnsureSelfSigned_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "server.crt")
	key := filepath.Join(dir, "server.key")

	if err := EnsureSelfSigned(cert, key); err != nil {
		t.Fatalf("first: %v", err)
	}
	first, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	// A second call must not overwrite the existing cert.
	if err := EnsureSelfSigned(cert, key); err != nil {
		t.Fatalf("second: %v", err)
	}
	second, _ := tls.LoadX509KeyPair(cert, key)
	if string(first.Certificate[0]) != string(second.Certificate[0]) {
		t.Fatal("existing cert was overwritten on the second call")
	}
}
