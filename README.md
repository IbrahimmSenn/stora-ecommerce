# I Love Shopping

A full-stack e-commerce platform built with Go, PostgreSQL, RabbitMQ, and Docker. Covers the full commerce loop — catalog browsing, guest and persistent carts, single-page checkout, Stripe sandbox payments, webhook-driven order state, async email over a message queue, order history, and the cancellation + refund workflow — with the order PII (contact + shipping address) encrypted at rest. Frontend is a React 19 + TypeScript storefront with a custom design system (OKLCH tokens, variable fonts, light/dark toggle, signature cart transition).

## Quick start

Prerequisite: **Docker** + **Docker Compose**. That's it.

```bash
git clone https://gitea.kood.tech/ibrahimsen/i-love-shopping.git
cd i-love-shopping
cp .env.example .env       # fill in Stripe + ENCRYPTION_KEY, others optional
make up                    # boots db, runs migrations, seeds, starts API
```

Open [http://localhost:8080](http://localhost:8080). To exercise Stripe end-to-end, run `stripe listen --forward-to http://localhost:8080/api/v1/webhooks/stripe` in a second terminal and paste the printed `whsec_...` into `.env`.

### Bundled services

| Service | URL | Purpose |
|---|---|---|
| API + storefront | [http://localhost:8080](http://localhost:8080) | Go API and built React app |
| PostgreSQL | `localhost:5433` | Application database |
| RabbitMQ | [localhost:15672](http://localhost:15672) (`guest`/`guest`) | Inspect `payments.emails` and the DLQ |
| Mailhog | [localhost:8025](http://localhost:8025) | Captures every outgoing email |

### Seed accounts

| Email | Password | Role |
|---|---|---|
| `admin@shop.com` | `admin123` | admin |
| `customer@shop.com` | `customer123` | customer |

Plus a small mixed catalog (7 categories, 5 brands, 10 products, reviews).

### Make targets

`make up` / `make down` / `make reset` (fresh DB) / `make test` / `make build` / `make migrate-up` / `make migrate-down`.

## Frontend

React 19 + TypeScript + Vite + Tailwind v4 storefront served at `/`. Header links: **Shop**, **Cart**, **Orders**, **Account** when logged in, **Admin** when the user has the admin role. The theme toggle on the far right cycles light / dark.

Routes:

| Path | What it covers |
|---|---|
| `/` | Catalog with featured tile + asymmetric grid |
| `/cart`, `/checkout`, `/orders/:id/pay`, `/orders/:id/confirmation` | Cart → checkout → Stripe → confirmation |
| `/orders`, `/orders/:id` | Order history (filter by status + date) and detail |
| `/register`, `/login` (optional 2FA prompt), `/forgot-password`, `/reset-password` | Account creation + recovery |
| `/auth/oauth/callback` | Lands here after Google / Facebook OAuth |
| `/account`, `/account/2fa/setup`, `/account/2fa/disable` | Profile + TOTP management |
| `/admin/products`, `/admin/categories`, `/admin/brands` | Admin CRUD (role-gated) |
| `/dev/tokens` | Token rotation + replay-detection tester |

### Commerce walkthrough

1. Browse the catalog. Add to cart — the **signature cart panel** slides in from the right; the nav cart count pulses.
2. Log in mid-shopping. If both a guest cart and a user cart exist, you'll be prompted to merge or keep one.
3. `/checkout` — single page, contact (prefilled when authed), address, shipping method. Submitting creates the order in `pending_payment` and reserves stock.
4. Stripe Elements at `/orders/:id/pay`. Test cards: `4242 4242 4242 4242` (succeeds), `4000 0000 0000 0002` (decline), `4000 0000 0000 9995` (insufficient funds).
5. After success, the confirmation page polls until the webhook flips the order to `paid`. The email lands in Mailhog.
6. From `/orders`, cancel a `paid` order — Stripe issues an idempotent refund, stock restocks, status flips to `refunded`, and a refund email lands in Mailhog.
7. The RabbitMQ UI shows `payments.emails` incrementing on each event; stop Mailhog mid-payment and the message ends up in `payments.emails.dlq` after three retries.

Access tokens live **in memory only** — refreshing the tab clears authentication. `/dev/tokens` exposes the rotation flow and demonstrates refresh-token replay detection.

### Theming

The design system lives in [web/src/styles/tokens.css](web/src/styles/tokens.css) and [web/src/index.css](web/src/index.css). Near-monochrome neutrals tinted toward an **oxblood** accent, all in OKLCH. Display face is **Bricolage Grotesque Variable**, body is **Hanken Grotesk Variable** — both self-hosted via `@fontsource-variable`. Light/dark choice persists in `localStorage`; the first load reads `prefers-color-scheme`. Motion (entrance reveals, cart panel slide, count pulse) collapses to instant when `prefers-reduced-motion: reduce` is set.

## API Reference

### Auth

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/auth/register` | Email + password + captcha token |
| POST | `/api/v1/auth/login` | Returns access + refresh tokens; may require `totp_code` |
| POST | `/api/v1/auth/refresh` | Single-use refresh token rotation |
| POST | `/api/v1/auth/logout` | Bearer — revoke all sessions |
| POST | `/api/v1/auth/forgot-password` | Request reset email |
| POST | `/api/v1/auth/reset-password` | Redeem reset token |
| GET | `/api/v1/auth/oauth/{provider}` | Redirect to Google/Facebook consent |
| GET | `/api/v1/auth/oauth/{provider}/callback` | OAuth completion |
| POST | `/api/v1/auth/2fa/setup` / `enable` / `disable` | Bearer — TOTP lifecycle |

### Catalog (public)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/products` | Search: `q`, `category_id`, `brand_id`, `min_price`, `max_price`, `min_rating`, `sort`, `page`, `page_size` |
| GET | `/api/v1/products/suggest?q=` | Typeahead |
| GET | `/api/v1/products/{id}` | Detail with images and reviews |
| GET | `/api/v1/categories` / `/categories/{slug}` | Category tree / by slug |
| GET | `/api/v1/brands` / `/brands/{id}` | Brands |

### Cart, checkout, orders

All accept a bearer token OR a `guest_session` cookie (issued automatically on first cart interaction).

| Method | Endpoint | Description |
|---|---|---|
| GET / POST / PUT / DELETE | `/api/v1/cart`, `/cart/items`, `/cart/items/{productId}` | CRUD on cart lines (409 on insufficient stock) |
| GET | `/api/v1/cart/merge-status` | Logged-in only — describes what a merge would do |
| POST | `/api/v1/cart/merge` | Logged-in only — body `{strategy: "merge"\|"keep_user"\|"keep_guest"}` |
| POST | `/api/v1/checkout` | Creates order in `pending_payment`, reserves stock |
| GET | `/api/v1/orders` | Owner-scoped; filter by `status`, `from`, `to` (RFC3339) |
| GET | `/api/v1/orders/{id}` | Detail with decrypted address |
| POST | `/api/v1/orders/{id}/cancel` | Unpaid → `cancelled` (restocks). Paid → idempotent Stripe refund → `refunded` (restocks) |

Statuses: `pending_payment` → `paid` → `processing` → `shipped` → `delivered`, plus terminal `payment_failed` (retryable), `cancelled`, `refunded`.

### Payments

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/orders/{id}/payment-intent` | Owner-checked — lazily creates a Stripe PaymentIntent, persists a `payments` row, returns `client_secret` + `publishable_key`. Safe to call after `payment_failed`. |
| POST | `/api/v1/webhooks/stripe` | Signature-verified. Handles `payment_intent.succeeded` and `payment_intent.payment_failed` — flips the order, persists payment metadata, publishes a JSON event to RabbitMQ. Idempotent. |
| GET | `/api/v1/config/stripe` | Returns the publishable key for the frontend |

### Admin (admin role required)

`POST /api/v1/admin/products`, `PUT /api/v1/admin/products/{id}`, `DELETE /api/v1/admin/products/{id}`, `POST /api/v1/admin/products/{id}/images`, `DELETE /api/v1/admin/products/{id}/images/{imageId}`, `POST /api/v1/admin/categories`, `POST /api/v1/admin/brands`.

### Messaging

Stripe webhook publishes to the `payments` topic exchange. A `notifications.email` consumer subscribes to `payments.emails` (bound to `payment.*`) and sends mail via SMTP. Failures retry in-process (200ms / 1s / 5s), then move to `payments.emails.dlq` via the `payments.dlx` fanout exchange.

| Routing key | Body |
|---|---|
| `payment.succeeded` | `{order_id, payment_intent_id, amount_cents, currency}` |
| `payment.failed` | `{order_id, payment_intent_id, amount_cents, currency, failure_code, failure_message}` |

## Architecture

```
HTTP → Handler (decode/validate) → Service (business logic) → Repository (SQL) → PostgreSQL
                                                            ↘ Event Publisher → RabbitMQ → Notifications Consumer → Mailer → SMTP
```

Each layer talks through Go interfaces, so each service has mock-based unit tests. Raw SQL via pgx — no ORM. Refunds avoid an `orders ↔ payments` import cycle by passing a `RefunderFunc` adapter wired in `cmd/api/main.go`. The Stripe webhook updates the database synchronously (the order is in `paid` before the webhook returns 200) and publishes the email as a side effect.

### PII encryption at rest

Order contact (email, phone) and every field of the shipping address are stored as `bytea` ciphertext — AES-256-GCM with a 32-byte key from `ENCRYPTION_KEY`. Each value carries its own nonce; plaintext never lands in Postgres. Verify it after placing an order:

```bash
docker exec -it $(docker ps -qf name=db) psql -U admin -d mystore -c \
  "SELECT order_number, encode(email_encrypted, 'hex') FROM orders ORDER BY created_at DESC LIMIT 1;"
```

## Testing

```bash
make test        # full Go test suite
cd web && npm run build   # type-check + production build
```

What's covered:

- **Commerce services** — cart add/update/remove + merge strategies; orders checkout (rejects empty cart, stock-changed conflict, encrypted PII round-trip, ownership checks, cancel restocks, cancel-of-paid triggers refund, refund failure leaves status paid); payments (rejects non-payable status, persists intent, webhook flips + publishes event, idempotent on retries, bad signature rejected, refund is idempotent).
- **Notifications consumer** — succeeded routes to confirmation body, failed includes reason + code, terminal errors on bad payloads, retry semantics.
- **Identity / catalog (from Project 1)** — JWT, login/refresh/logout/2FA/password-reset, category tree, product CRUD, middleware enforcement, security (SQL injection, XSS, JWT tampering, oversized bodies, malformed JSON, negative-value injection).

### Concurrent payment guard

With one unit in stock, open two sessions and check the same product out in parallel — the second checkout fails with `409 stock or price changed while you were checking out`. `SELECT ... FOR UPDATE` on every cart line serialises the stock decrement.

## Environment

`.env.example` lists every variable; only a handful are strictly required for a working stack:

- `DATABASE_URL`, `JWT_SECRET`, `ENCRYPTION_KEY` (generate with `openssl rand -hex 32`), `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PUBLISHABLE_KEY`, `RABBITMQ_URL`.

Optional (features degrade gracefully if absent): `GOOGLE_CLIENT_ID/SECRET`, `FB_CLIENT_ID/SECRET`, `RECAPTCHA_SITE_KEY/SECRET_KEY` (set `SKIP_CAPTCHA=true` in dev), `SMTP_HOST/PORT/USER/PASS/FROM` (Mailhog handles this in compose).

`docker-compose.yml` overrides `DATABASE_URL`, `RABBITMQ_URL`, `SMTP_HOST`, `SMTP_PORT` so a `.env` copied verbatim from `.env.example` (with its localhost defaults) works unchanged inside the container network.

## Project layout

```
cmd/api/main.go         entrypoint, dependency wiring, graceful shutdown
internal/
  auth/                 login, refresh, 2FA, password reset
  brand/  category/  product/   catalog (faceted search + admin CRUD)
  captcha/  oauth/      reCAPTCHA v3 + Google/Facebook OAuth
  cart/                 guest + persistent carts, merge
  checkout/  orders/    single-page checkout, history, cancel + refund
  payments/             Stripe intents, webhook, refunder, event publisher
  notifications/        consumer that turns payment events into emails
  messaging/            RabbitMQ connection, publisher, consumer, topology
  mailer/               SMTP sender (Mailhog-compatible)
  crypto/               AES-256-GCM encryptor for order PII
  middleware/           auth, admin role, optional auth, guest session
  ctxkey/  response/    shared context keys + JSON response helpers
migrations/             21 migrations + seed.sql
web/                    React 19 + TypeScript + Vite + Tailwind v4 storefront
Dockerfile              multi-stage: node → golang → alpine
docker-compose.yml      full stack: db + migrate + seed + rabbitmq + mailhog + api
Makefile                dev commands
```
