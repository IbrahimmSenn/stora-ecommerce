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

Tokens in [web/src/styles/tokens.css](web/src/styles/tokens.css) — near-monochrome OKLCH neutrals tinted toward an **oxblood** accent. Display face **Bricolage Grotesque Variable**, body **Hanken Grotesk Variable**, both self-hosted via `@fontsource-variable`. Light/dark persists in `localStorage` (first load reads `prefers-color-scheme`); motion collapses to instant under `prefers-reduced-motion: reduce`.

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
| GET | `/api/v1/recommendations` | Personalised picks from activity + current cart contents |

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
| GET | `/api/v1/config/stripe` / `/config/recaptcha` | Publishable keys for the frontend |

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

Each layer talks through Go interfaces — services are mock-tested in isolation. Raw SQL via pgx, no ORM. A `RefunderFunc` adapter wired in `cmd/api/main.go` breaks the `orders ↔ payments` cycle. The Stripe webhook updates the database synchronously (order is `paid` before the webhook returns 200); the confirmation email is a side effect over RabbitMQ.

### Data model

Commerce tables added on top of the Project 1 identity + catalog schema. The encrypted columns (`*_encrypted`, `*_enc`) are AES-256-GCM bytea — see [PII encryption at rest](#pii-encryption-at-rest).

```mermaid
erDiagram
    users {
        UUID id PK
    }
    products {
        UUID id PK
    }

    carts {
        UUID id PK
        UUID user_id FK "nullable, unique"
        UUID guest_session_id "nullable, unique"
    }
    cart_items {
        UUID id PK
        UUID cart_id FK
        UUID product_id FK
        INT quantity
    }
    orders {
        UUID id PK
        VARCHAR order_number UK
        UUID user_id FK "nullable"
        UUID guest_session_id "nullable"
        VARCHAR status "pending_payment..refunded"
        BYTEA email_encrypted
        BYTEA phone_encrypted
        INT subtotal_cents
        INT shipping_cents
        INT total_cents
        VARCHAR shipping_method
    }
    order_items {
        UUID id PK
        UUID order_id FK
        UUID product_id FK "nullable"
        VARCHAR product_name "snapshot"
        INT unit_price_cents
        INT quantity
    }
    shipping_addresses {
        UUID id PK
        UUID order_id FK,UK
        BYTEA recipient_name_encrypted
        BYTEA line1_encrypted
        BYTEA line2_encrypted
        BYTEA city_encrypted
        BYTEA region_encrypted
        BYTEA postal_code_encrypted
        BYTEA country_encrypted
    }
    payments {
        UUID id PK
        UUID order_id FK
        BYTEA stripe_payment_intent_id_enc
        BYTEA stripe_payment_intent_id_hmac UK
        BYTEA stripe_refund_id_enc
        VARCHAR status "pending..refunded"
        INT amount_cents
        VARCHAR currency
        BYTEA error_code_enc
        BYTEA error_message_enc
        TIMESTAMPTZ refunded_at
    }

    users ||--o| carts : "owns"
    users ||--o{ orders : "places"
    carts ||--o{ cart_items : "contains"
    products ||--o{ cart_items : "appears in"
    orders ||--o{ order_items : "lists"
    products ||--o{ order_items : "snapshotted by"
    orders ||--|| shipping_addresses : "ships to"
    orders ||--o{ payments : "settled by"
```

Full schema including auth and catalog tables in [docs/erd.mmd](docs/erd.mmd).

### PII encryption at rest

Order contact (email, phone) and every field of the shipping address are stored as `bytea` ciphertext — AES-256-GCM with a 32-byte key from `ENCRYPTION_KEY`. Each value carries its own nonce; plaintext never lands in Postgres. Verify it after placing an order:

```bash
docker exec -it $(docker ps -qf name=db) psql -U admin -d mystore -c \
  "SELECT order_number, encode(email_encrypted, 'hex') FROM orders ORDER BY created_at DESC LIMIT 1;"
```

## Testing

```bash
make test                 # full Go test suite
cd web && npm run build   # type-check + production build
cd web && npm test        # Vitest frontend suite
```

What's covered:

- **Commerce services** — cart add/update/remove + merge strategies; orders checkout (rejects empty cart, stock-changed conflict, encrypted PII round-trip, ownership checks, cancel restocks, cancel-of-paid triggers refund, refund failure leaves status paid); payments (rejects non-payable status, persists intent, webhook flips + publishes event, idempotent on retries, bad signature rejected, refund is idempotent).
- **Notifications consumer** — succeeded routes to confirmation body, failed includes reason + code, terminal errors on bad payloads, retry semantics.
- **Identity / catalog (from Project 1)** — JWT, login/refresh/logout/2FA/password-reset, category tree, product CRUD, middleware enforcement, security (SQL injection, XSS, JWT tampering, oversized bodies, malformed JSON, negative-value injection).
- **Frontend (Vitest + Testing Library)** — checkout form validation (required fields, email + phone + postal + country code, whitespace trimming) and the recommendations rail (loading, empty, render, error resilience).

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
  auth/  user/          login, refresh, 2FA, password reset, profile
  brand/  category/  product/   catalog (faceted search + admin CRUD)
  captcha/  oauth/      reCAPTCHA v3 + Google/Facebook OAuth
  cart/                 guest + persistent carts, merge
  orders/               single-page checkout, history, cancel + refund
  payments/             Stripe intents, webhook, refunder, event publisher
  notifications/        consumer that turns payment events into emails
  messaging/            RabbitMQ connection, publisher, consumer, topology
  mailer/               SMTP sender (Mailhog-compatible)
  crypto/               AES-256-GCM encryptor for order PII
  activity/  recommend/ event log + cart-aware recommendation service
  middleware/           auth, admin role, optional auth, guest session
  config/               env loading
  ctxkey/  response/    shared context keys + JSON response helpers
migrations/             25 migrations + seed.sql
web/                    React 19 + TypeScript + Vite + Tailwind v4 storefront
docs/erd.mmd            full entity-relationship diagram
Dockerfile              multi-stage: node → golang → alpine
docker-compose.yml      full stack: db + migrate + seed + rabbitmq + mailhog + api
Makefile                dev commands
```
