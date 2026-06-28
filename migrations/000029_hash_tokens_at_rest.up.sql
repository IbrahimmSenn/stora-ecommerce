-- Tokens are now stored as SHA-256 digests (see internal/auth/hash.go). Clear
-- any pre-existing plaintext rows so none linger; these tables are transient,
-- so the only effect is that active sessions/reset links must be re-issued.
DELETE FROM refresh_tokens;
DELETE FROM password_reset_tokens;
