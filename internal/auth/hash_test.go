package auth

import "testing"

func TestHashToken_DeterministicAndOpaque(t *testing.T) {
	tok := "header.payload.signature"

	h1 := hashToken(tok)
	h2 := hashToken(tok)
	if h1 != h2 {
		t.Fatal("hash must be deterministic so lookups match")
	}
	if h1 == tok {
		t.Fatal("hash must not equal the raw token")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars (sha-256), got %d", len(h1))
	}
}

func TestHashToken_DistinctInputsDiffer(t *testing.T) {
	if hashToken("a.b.c") == hashToken("a.b.d") {
		t.Fatal("different tokens must hash differently")
	}
}

func TestHashToken_Empty(t *testing.T) {
	if hashToken("") != "" {
		t.Fatal("empty token should hash to empty string")
	}
}
