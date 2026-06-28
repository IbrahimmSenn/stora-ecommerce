// hash.go — at-rest hashing for bearer-style tokens (refresh + password reset).
//
// These tokens are high-entropy secrets the server only ever looks up by exact
// value, so we store a SHA-256 digest instead of the raw token. A database leak
// then exposes only digests, which are useless to an attacker — the same
// rationale as hashing passwords, minus the need for a slow KDF (the tokens are
// already unguessable, so brute force is infeasible).
package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashToken returns the hex SHA-256 of a token. Empty in, empty out.
func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
