# Security model — CIA triad & data protection

How confidentiality, integrity, and availability are enforced in this codebase,
and exactly which data is protected by which mechanism.

## Data protection at rest

| Data | Mechanism | Where |
|---|---|---|
| Passwords | bcrypt (one-way, salted, cost-tuned) | `internal/user`, `internal/auth` |
| User email | AES-256-GCM + HMAC-SHA256 blind index (`email_hmac`) for login/uniqueness | `internal/user/repository.go`, migration `000030` |
| Order contact email / phone | AES-256-GCM | `internal/orders` (`email_encrypted`, `phone_encrypted`) |
| Shipping addresses (name/line1/line2/city/region/postal/country) | AES-256-GCM | `internal/orders` (`shipping_addresses.*_encrypted`) |
| Payment identifiers (Stripe intent id, refund id, error fields) | AES-256-GCM + HMAC blind index for lookup | `internal/payments`, migration `000022` |
| Refresh tokens | SHA-256 digest (look-up by hash) | `internal/auth/hash.go`, migration `000029` |
| Password-reset tokens | SHA-256 digest | `internal/auth/hash.go` |
| 2FA TOTP secret + recovery codes | AES-256-GCM (hex-wrapped in the existing column) | `internal/auth/repository.go` (`encField`/`decField`) |
| Card numbers / CVV / expiry | **never stored** — Stripe Elements tokenises client-side | n/a (PCI-DSS) |

Crypto primitives live in `internal/crypto` (AES-256-GCM with a random nonce per
value; HMAC-SHA256 for deterministic equality lookups on encrypted columns). The
key is supplied via `ENCRYPTION_KEY` (32 bytes / 64 hex) and validated at boot.

User email is normalised (lower-cased, trimmed) before hashing so the blind
index is stable, which also makes login case-insensitive. Demo users are seeded
by the app at startup (`internal/seed`) since AES-GCM ciphertext can't be
produced in `seed.sql`.

## Confidentiality
- Encryption at rest (table above).
- TLS in transit — HTTPS on `:8443` with a self-signed cert generated in-process
  at boot (`internal/tlsutil`, no openssl/manual step); HTTP stays on `:8080`.
  Cookies are marked `Secure` when `APP_ENV=production`.
- Access control: JWT access tokens held **in memory only** (never localStorage),
  HttpOnly refresh cookie, RBAC (`admin`/`support`/`sales`/`customer`) with
  least-privilege route groups, mandatory 2FA for staff.
- CORS restricted to configured origins; no wildcard-with-credentials.

## Integrity
- AES-GCM is authenticated encryption — any tampering with ciphertext fails
  decryption rather than yielding altered plaintext.
- Refresh-token rotation: single-use, replay detected (a reused token revokes the
  whole chain).
- Admin mutations are recorded to an append-only `admin_audit_log`.
- Stripe webhooks are signature-verified before they can change order state.
- ACID transactions guard multi-step writes (checkout stock lock, refunds);
  inventory uses `SELECT … FOR UPDATE` so two buyers can't oversell the last unit.
- Server- and client-side input validation; parameterised SQL (pgx) throughout.

## Availability
- Token-bucket rate limiting per IP (strict on auth endpoints) returns `429`.
- HTTP server timeouts (read/write/idle/header) blunt slow-loris.
- Graceful shutdown drains the HTTP server and the RabbitMQ consumer.
- RabbitMQ dead-letter queue + retry for transient email/consumer failures.
- Container healthcheck + `restart: unless-stopped`.
