-- Encrypt user email at rest. The plaintext `email` column (and its UNIQUE
-- constraint) is replaced by:
--   email_encrypted — AES-256-GCM ciphertext (for display, decrypted in-app)
--   email_hmac      — deterministic HMAC-SHA256 blind index (for equality
--                     lookup at login + uniqueness), written by the app.
--
-- Note: dropping `email` discards any existing plaintext addresses. This project
-- rebuilds from scratch (`make reset`) and re-seeds demo users through the app
-- (which encrypts), so there is no data to migrate here.
ALTER TABLE users ADD COLUMN email_encrypted BYTEA;
ALTER TABLE users ADD COLUMN email_hmac BYTEA;

ALTER TABLE users DROP COLUMN email; -- also drops the UNIQUE(email) constraint

CREATE UNIQUE INDEX users_email_hmac_key ON users (email_hmac);
