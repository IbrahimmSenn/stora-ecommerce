-- Display name is PII, so it's stored AES-encrypted like the email. No HMAC
-- blind index: nothing looks users up by name.
ALTER TABLE users ADD COLUMN name_encrypted BYTEA;
