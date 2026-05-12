# I Love Shopping

A full-scale B2C e-commerce platform built with Go, PostgreSQL, RabbitMQ, and Docker. Covers the full commerce loop: catalog browsing, guest and persistent carts, single-page checkout, Stripe sandbox payments with secure Elements, webhook-driven order state, asynchronous email notifications over a message queue, order history with filtering, and the cancellation + refund workflow. Sensitive PII (order contact details and shipping addresses) is encrypted at rest with AES-256-GCM. Ships with a React storefront and a built-in test panel so reviewers can exercise every flow in the browser.

## Features

### Commerce
- **Shopping Cart**: Guest carts (cookie-bound session) and persistent carts (DB-backed for logged-in users), with item add/update/remove, real-time totals, out-of-stock guards, and a merge prompt on login.
- **Checkout**: Single-page flow capturing contact, shipping address (format-validated), shipping method (flat-rate `standard` / `express`), and order summary; logged-in users get their email prefilled.
- **Stripe Payments**: Stripe Elements integration (never touches raw card data), lazy `PaymentIntent` creation per attempt, idempotent webhook for `payment_intent.succeeded` and `payment_intent.payment_failed`.
- **Inventory locking**: `SELECT ... FOR UPDATE` on every cart line at checkout prevents overselling under concurrent payments.
- **Encryption at rest**: AES-256-GCM (key from `ENCRYPTION_KEY`, 64-hex-char) on order email, phone, and every shipping-address field. Payment intent IDs are Stripe pointers, not PII.
- **Order Management**: History list filtered by status + date range, detail view with status, item-level pricing, and shipping breakdown. Cancellation allowed for unprocessed orders (`pending_payment` and `paid`). **Refund workflow**: cancelling a paid order triggers an idempotent Stripe refund, restocks inventory, and flips status to `refunded`.
- **Async Email Notifications**: Payment webhook publishes events to a RabbitMQ topic exchange (`payments`); a `notifications` consumer subscribes to `payments.emails`, loads the order, and sends confirmation or failure email via SMTP. Three in-process retries with exponential backoff, then DLX → `payments.emails.dlq` for inspection.
- **Mailhog**: Bundled SMTP sink with a web UI so reviewers can see every outgoing email without configuring real SMTP.

### Identity & Catalog (Project 1)
- **Authentication**: Email/password registration and login with JWT (access + refresh tokens)
- **OAuth**: Google and Facebook social login
- **CAPTCHA**: Google reCAPTCHA v3 on registration
- **2FA**: Optional TOTP-based two-factor authentication with QR codes and recovery codes
- **Password Recovery**: Email-based password reset with secure one-time tokens
- **Refresh Token Rotation**: Single-use refresh tokens with replay detection
- **Product Catalog**: Full CRUD with faceted search, sorting, and pagination
- **Categories**: Hierarchical tree structure with nested browsing
- **Search**: PostgreSQL full-text search (tsvector/GIN index) with weighted ranking
- **Role-Based Access**: Customer and admin roles with middleware enforcement

### Infrastructure
- **Docker**: Fully containerized — Docker and the Stripe CLI are the only host prerequisites
- **Seed Data**: Pre-loaded admin/customer accounts, categories, brands, products, and reviews

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| Router | Chi v5 |
| Database | PostgreSQL 16 |
| Message Queue | RabbitMQ 3.13 (amqp091-go) |
| Payments | Stripe sandbox (`stripe-go/v76` + `@stripe/react-stripe-js`) |
| Email (dev) | Mailhog (SMTP + web UI) |
| Auth | JWT (HS256), bcrypt, TOTP (pquerna/otp) |
| OAuth | golang.org/x/oauth2 (Google, Facebook) |
| Migrations | golang-migrate |
| Validation | go-playground/validator |
| Encryption | AES-256-GCM (`crypto/aes`, `crypto/cipher`) |
| Containers | Docker, docker-compose |
| Frontend | React 19 + TypeScript + Vite + Tailwind v4 |

## Entity Relationship Diagram

```mermaid
erDiagram
    users {
        UUID id PK
        VARCHAR email UK
        VARCHAR password_hash
        VARCHAR role
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    categories {
        UUID id PK
        VARCHAR name UK
        VARCHAR slug UK
        UUID parent_id FK
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    brands {
        UUID id PK
        VARCHAR name UK
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    products {
        UUID id PK
        VARCHAR name
        TEXT description
        INT price
        INT stock_quantity
        UUID category_id FK
        UUID brand_id FK
        INT weight_g
        INT weight_oz "generated"
        NUMERIC dimensions_cm
        NUMERIC dimensions_inch "generated"
        TSVECTOR search_vector
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    product_images {
        UUID id PK
        UUID product_id FK
        TEXT url
        BOOLEAN is_primary
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    reviews {
        UUID id PK
        UUID user_id FK
        UUID product_id FK
        TEXT comment
        SMALLINT rating
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    refresh_tokens {
        UUID id PK
        VARCHAR token UK
        UUID user_id FK
        BOOLEAN revoked
        BOOLEAN used
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
        TIMESTAMPTZ expires_at
    }

    oauth_accounts {
        UUID id PK
        UUID user_id FK
        VARCHAR provider
        VARCHAR provider_user_id
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    password_reset_tokens {
        UUID id PK
        UUID user_id FK
        VARCHAR token UK
        BOOLEAN used
        TIMESTAMPTZ created_at
        TIMESTAMPTZ expires_at
    }

    two_factor_auth {
        UUID id PK
        UUID user_id FK
        TEXT secret_key
        BOOLEAN is_enabled
        TEXT[] recovery_codes
        TIMESTAMPTZ created_at
    }

    carts {
        UUID id PK
        UUID user_id FK
        UUID guest_session_id
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    cart_items {
        UUID id PK
        UUID cart_id FK
        UUID product_id FK
        INT quantity
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    orders {
        UUID id PK
        VARCHAR order_number UK
        UUID user_id FK
        UUID guest_session_id
        VARCHAR status
        BYTEA email_encrypted
        BYTEA phone_encrypted
        INT subtotal_cents
        INT shipping_cents
        INT total_cents
        VARCHAR shipping_method
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    order_items {
        UUID id PK
        UUID order_id FK
        UUID product_id FK
        VARCHAR product_name
        INT unit_price_cents
        INT quantity
        TIMESTAMPTZ created_at
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
        TIMESTAMPTZ created_at
    }

    payments {
        UUID id PK
        UUID order_id FK
        TEXT stripe_payment_intent_id UK
        VARCHAR status
        INT amount_cents
        VARCHAR currency
        TEXT error_code
        TEXT error_message
        TEXT stripe_refund_id
        TIMESTAMPTZ refunded_at
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    users ||--o{ refresh_tokens : "has"
    users ||--o{ oauth_accounts : "links"
    users ||--o{ password_reset_tokens : "requests"
    users ||--o| two_factor_auth : "configures"
    users ||--o{ reviews : "writes"
    users ||--o| carts : "owns (logged-in)"
    users ||--o{ orders : "places"

    categories ||--o{ categories : "parent of"
    categories ||--o{ products : "contains"
    brands ||--o{ products : "manufactures"

    products ||--o{ product_images : "has"
    products ||--o{ reviews : "receives"
    products ||--o{ cart_items : "queued in"
    products ||--o{ order_items : "purchased as"

    carts ||--o{ cart_items : "contains"
    orders ||--o{ order_items : "contains"
    orders ||--|| shipping_addresses : "ships to"
    orders ||--o{ payments : "charged via"
```

### Relationships Summary

| Relationship | Cardinality | Description |
|---|---|---|
| users → refresh_tokens | 1:N | A user can have many refresh tokens (multiple sessions) |
| users → oauth_accounts | 1:N | A user can link multiple OAuth providers |
| users → password_reset_tokens | 1:N | A user can request multiple resets |
| users → two_factor_auth | 1:0..1 | A user can optionally enable 2FA |
| users → reviews | 1:N | A user can write many reviews |
| users → carts | 1:0..1 | At most one persistent cart per user (partial unique index) |
| users → orders | 1:N | Order history is preserved per user |
| categories → categories | 1:N (self) | Categories form a tree (parent_id) |
| categories → products | 1:N | A category contains many products |
| brands → products | 1:N | A brand has many products |
| products → product_images | 1:N | A product has many images |
| products → reviews | 1:N | A product receives many reviews |
| reviews (user_id, product_id) | UNIQUE | One review per user per product |
| carts → cart_items | 1:N | Each line is a unique product within the cart |
| cart_items (cart_id, product_id) | UNIQUE | Adding the same product twice updates quantity instead |
| orders → order_items | 1:N | Items snapshot price + name at order time so price changes don't rewrite history |
| orders → shipping_addresses | 1:1 | Each order has exactly one shipping address (encrypted) |
| orders → payments | 1:N | Each retry adds a fresh `payments` row; the latest drives order status |

**Encryption note:** `*_encrypted` columns are AES-256-GCM ciphertext (nonce + ciphertext + auth tag concatenated). Plaintext never lands in Postgres; decryption happens in the Go service layer using `ENCRYPTION_KEY`.

## Setup

### Prerequisites

- **Docker** and **Docker Compose** (only host requirements — all other dependencies are managed within containers)

### Quick Start

1. Clone the repository:
   ```bash
   git clone https://gitea.kood.tech/ibrahimsen/i-love-shopping.git
   cd i-love-shopping
   ```

2. Copy and configure the environment file:
   ```bash
   cp .env.example .env
   ```
   Edit `.env` to add your credentials for OAuth, reCAPTCHA, and SMTP (see [Environment Variables](#environment-variables) below). All features work without these — they simply enable the optional integrations.

3. Start everything:
   ```bash
   make up
   ```
   This starts PostgreSQL, runs all 14 migrations, seeds the database with sample data, and launches the API on port **8080**.

4. Open the test panel: [http://localhost:8080](http://localhost:8080)

5. Verify the API:
   ```bash
   curl http://localhost:8080/health
   # {"status":"ok"}
   ```

### Bundled dev services

`make up` brings up four supporting containers alongside the API:

| Service | URL | Purpose |
|---------|-----|---------|
| PostgreSQL | `localhost:5433` | Application database |
| RabbitMQ | `localhost:5672` (AMQP), [localhost:15672](http://localhost:15672) (management UI, `guest`/`guest`) | Payment event bus; inspect the `payments.emails` queue and the `payments.emails.dlq` dead-letter queue |
| Mailhog | [localhost:8025](http://localhost:8025) (web UI), `localhost:1025` (SMTP) | Captures all outgoing email so you can read confirmations, failures, and password resets without configuring real SMTP |

For Stripe webhooks, run `stripe listen --forward-to http://localhost:8080/api/v1/webhooks/stripe` in a separate terminal and copy the printed `whsec_...` into `STRIPE_WEBHOOK_SECRET`.

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make up` | Build and start all Docker services |
| `make down` | Stop all services |
| `make reset` | Stop, delete database volume, rebuild and restart (fresh state) |
| `make run` | Run the API locally (requires local Go and PostgreSQL) |
| `make build` | Build the Go binary |
| `make test` | Run all automated tests |
| `make migrate-up` | Apply migrations (local development) |
| `make migrate-down` | Roll back migrations (local development) |

### Seed Data

The database is automatically seeded on startup with test data:

| Email | Password | Role |
|-------|----------|------|
| `admin@shop.com` | `admin123` | **admin** |
| `customer@shop.com` | `customer123` | customer |

Plus 7 categories (hierarchical), 5 brands, 10 products with images, and 9 reviews with ratings.

### Environment Variables

All credentials are loaded from the `.env` file (not committed to the repository). Copy `.env.example` and fill in the values you need.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | Secret for signing JWTs |
| `PORT` | No | `8080` | API server port |
| `BASE_URL` | No | `http://localhost:8080` | Public base URL (for OAuth callbacks, reset links) |
| `BCRYPT_COST` | No | `10` | bcrypt hashing cost |
| `GOOGLE_CLIENT_ID` | No | — | Google OAuth client ID ([console.cloud.google.com](https://console.cloud.google.com/)) |
| `GOOGLE_CLIENT_SECRET` | No | — | Google OAuth client secret |
| `FB_CLIENT_ID` | No | — | Facebook OAuth client ID |
| `FB_CLIENT_SECRET` | No | — | Facebook OAuth client secret |
| `RECAPTCHA_SITE_KEY` | No | — | reCAPTCHA v3 site key ([google.com/recaptcha/admin](https://www.google.com/recaptcha/admin)) |
| `RECAPTCHA_SECRET_KEY` | No | — | reCAPTCHA v3 secret key |
| `SKIP_CAPTCHA` | No | `false` | Set `true` to skip CAPTCHA in development |
| `SMTP_HOST` | No | — | SMTP server host (empty = skip emails). `docker-compose` overrides to `mailhog`. |
| `SMTP_PORT` | No | `587` | SMTP port. `docker-compose` overrides to `1025` for Mailhog. |
| `SMTP_USER` | No | — | SMTP username (leave empty for Mailhog — it accepts unauthenticated SMTP) |
| `SMTP_PASS` | No | — | SMTP password |
| `SMTP_FROM` | No | (SMTP_USER) | Sender email address |
| `STRIPE_SECRET_KEY` | Yes | — | Stripe sandbox secret (`sk_test_...`) |
| `STRIPE_WEBHOOK_SECRET` | Yes | — | Webhook signing secret (`whsec_...`) printed by `stripe listen` |
| `STRIPE_PUBLISHABLE_KEY` | Yes | — | Stripe publishable key (`pk_test_...`) used by the frontend |
| `ENCRYPTION_KEY` | Yes | — | 64-hex-char (32-byte) AES-256-GCM key for encrypting order PII at rest. Generate with `openssl rand -hex 32`. |
| `RABBITMQ_URL` | Yes | — | AMQP URL (e.g. `amqp://guest:guest@localhost:5672/`). `docker-compose` overrides to the `rabbitmq` service hostname. |

## Frontend

The React storefront is mounted at `http://localhost:8080`. Header links cover the public catalog, cart, order history, and account; an `Admin` link appears when logged in as `admin@shop.com`. A separate legacy test panel for identity flows (register, OAuth, 2FA, token rotation, password reset) is reachable from the header as **Test Panel** for reviewers who want raw access to those endpoints.

### Commerce walkthrough

1. **Browse** `/products` — search, faceted filters (category, brand, price, rating), sort, pagination, product detail with reviews.
2. **Build a cart** — Add items as a guest (a `guest_session` cookie is issued by the server; the cart persists across page reloads). Update quantity or remove items inline. The cart totals recompute live and the header badge reflects line count.
3. **Out-of-stock guard** — Use the Admin panel to set a product's `stock_quantity` to 0, then try to add it; the API returns a specific error message and the UI surfaces it.
4. **Log in mid-shopping** — Log in with a guest cart present. The app prompts to merge the guest items into your persistent cart. Decline to keep both isolated.
5. **Checkout** — `/checkout` is single-page: contact (email prefills if logged in), shipping address (RFC-style format validation), shipping method (`standard` $5 / `express` $15), order summary with itemised totals. Submit creates the order in `pending_payment` and reserves stock.
6. **Pay** — You're redirected to `/orders/{id}/pay` which mounts Stripe Elements. Use `4242 4242 4242 4242` for success, `4000 0000 0000 0002` for a generic decline, `4000 0000 0000 9995` for insufficient funds, `4000 0000 0000 0069` for expired card. Card number, expiry, and CVV are validated client-side by Stripe before submit. On success the page polls until the webhook flips the order to `paid`.
7. **Email confirmation** — Open the Mailhog UI at [http://localhost:8025](http://localhost:8025) — a "payment received" email is in the inbox. Failures also land here.
8. **Order history** — `/orders` lists past orders filterable by status and date range. Click into one for the detail view with the full status, items, shipping, and totals.
9. **Cancel + refund** — Cancel a `pending_payment` order: stock restocks, status flips to `cancelled`. Cancel a `paid` order: a Stripe refund is issued (idempotency-keyed), stock is restocked, and status flips to `refunded`. The cancel-confirm dialog adapts its copy to mention the refund.
10. **Watch the queue** — Open the RabbitMQ management UI at [http://localhost:15672](http://localhost:15672) (`guest`/`guest`). The `payments.emails` queue increments `messages_acknowledged` on every payment event. Stop Mailhog mid-payment and the message ends up in `payments.emails.dlq` after three retries.

Access tokens are stored **in JavaScript memory only** (not localStorage or cookies) — refreshing the page clears authentication, demonstrating proper in-memory token storage.

## API Reference

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/auth/register` | — | Register with email/password (+ captcha token) |
| POST | `/api/v1/auth/login` | — | Login (returns access + refresh tokens) |
| POST | `/api/v1/auth/refresh` | — | Rotate refresh token |
| POST | `/api/v1/auth/logout` | Bearer | Revoke all sessions |
| POST | `/api/v1/auth/forgot-password` | — | Request password reset email |
| POST | `/api/v1/auth/reset-password` | — | Reset password with token |

### OAuth

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/auth/oauth/{provider}` | Redirect to Google/Facebook consent screen |
| GET | `/api/v1/auth/oauth/{provider}/callback` | OAuth callback (redirects to frontend with tokens) |

### Two-Factor Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/auth/2fa/setup` | Bearer | Generate TOTP secret + QR code + recovery codes |
| POST | `/api/v1/auth/2fa/enable` | Bearer | Verify TOTP code to activate 2FA |
| POST | `/api/v1/auth/2fa/disable` | Bearer | Verify TOTP/recovery code to deactivate 2FA |

### Product Catalog (Public)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/products` | Search/filter products |
| GET | `/api/v1/products/suggest?q=` | Dynamic search suggestions (typeahead) |
| GET | `/api/v1/products/{id}` | Get product detail with images and reviews |
| GET | `/api/v1/categories` | Get category tree |
| GET | `/api/v1/categories/{slug}` | Get category by slug |
| GET | `/api/v1/brands` | List all brands |
| GET | `/api/v1/brands/{id}` | Get brand by ID |

**Search query parameters:**

| Param | Type | Example | Description |
|-------|------|---------|-------------|
| `q` | string | `wireless headphones` | Full-text search (tsvector) |
| `category_id` | UUID | `550e8400-...` | Filter by category |
| `brand_id` | UUID | `550e8400-...` | Filter by brand |
| `min_price` | int | `1000` | Min price in cents |
| `max_price` | int | `5000` | Max price in cents |
| `min_rating` | int | `4` | Minimum average rating |
| `sort` | string | `price_asc` | Sort: `relevance`, `price_asc`, `price_desc`, `rating` |
| `page` | int | `1` | Page number |
| `page_size` | int | `20` | Items per page (max 100) |

### Admin (Requires admin role)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/admin/products` | Create product |
| PUT | `/api/v1/admin/products/{id}` | Update product |
| DELETE | `/api/v1/admin/products/{id}` | Delete product |
| POST | `/api/v1/admin/products/{id}/images` | Add product image |
| DELETE | `/api/v1/admin/products/{id}/images/{imageId}` | Delete product image |
| POST | `/api/v1/admin/categories` | Create category |
| POST | `/api/v1/admin/brands` | Create brand |

### Shopping Cart

All cart endpoints accept either a bearer token (logged-in user) or a `guest_session` cookie (issued automatically on first cart interaction). Switching between the two is supported via the merge flow at the bottom of this table.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/cart` | Get the current cart with line items, per-line stock, and totals |
| POST | `/api/v1/cart/items` | Add a product (`{product_id, quantity}`); returns 409 if stock is insufficient |
| PUT | `/api/v1/cart/items/{productId}` | Set the quantity for a line; quantity `0` removes the line |
| DELETE | `/api/v1/cart/items/{productId}` | Remove a single line |
| DELETE | `/api/v1/cart` | Empty the cart |
| GET | `/api/v1/cart/merge-status` | Logged-in only. Reports whether a guest cart exists for the cookie and what would happen on merge |
| POST | `/api/v1/cart/merge` | Logged-in only. Body `{strategy: "merge"\|"keep_user"\|"keep_guest"}` consolidates the guest cart into the user cart |

### Checkout & Orders

Same auth surface as cart (token OR guest cookie). The checkout endpoint creates an order in `pending_payment` and decrements stock atomically.

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/checkout` | Place an order from the current cart. Body includes contact, shipping address, and shipping method (`standard` or `express`). Returns the created order. |
| GET | `/api/v1/orders` | List orders owned by the caller (user or guest). Query: `status`, `from` (RFC3339), `to` (RFC3339) |
| GET | `/api/v1/orders/{id}` | Order detail with items, decrypted shipping address, and current status |
| POST | `/api/v1/orders/{id}/cancel` | Cancel an unprocessed order. Paid orders trigger an idempotent Stripe refund and end in `refunded`; unpaid orders end in `cancelled`. Both restock inventory. |

**Order statuses:** `pending_payment` → `paid` → `processing` → `shipped` → `delivered`, plus terminal `payment_failed` (retryable), `cancelled`, `refunded`.

### Payments

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/orders/{id}/payment-intent` | Owner-checked | Lazily creates a Stripe `PaymentIntent` for the order and persists a fresh `payments` row in status `pending`. Returns `client_secret` + `publishable_key` for Stripe Elements. Safe to call again after a `payment_failed`. |
| POST | `/api/v1/webhooks/stripe` | Signature-verified | Stripe webhook endpoint. Handles `payment_intent.succeeded` and `payment_intent.payment_failed`, updates the payments row, flips the order, and publishes a JSON event to RabbitMQ. Idempotent — Stripe retries are no-ops. |
| GET | `/api/v1/config/stripe` | — | Returns the public Stripe publishable key for the frontend |

### Messaging Topology

The Stripe webhook publishes events on the `payments` topic exchange. The `notifications.email` consumer subscribes to `payments.emails` (bound to `payment.*`) and sends the email. On failure the message is retried in-process (200ms / 1s / 5s backoff), then NACK'd to the `payments.dlx` fanout exchange and held in `payments.emails.dlq` for inspection.

| Routing key | Body shape |
|---|---|
| `payment.succeeded` | `{order_id, payment_intent_id, amount_cents, currency}` |
| `payment.failed` | `{order_id, payment_intent_id, amount_cents, currency, failure_code, failure_message}` |

## Testing

### Automated Tests

Run the full test suite:
```bash
make test
```

**Commerce unit tests:**
- Cart service: add/update/remove, totals, out-of-stock guard, guest ↔ user merge strategies
- Orders service: empty-cart rejection, stock-changed conflict, encrypted PII round-trip on checkout, ownership checks on GetByID, cancel restocks + flips status, **cancel of a paid order calls the refunder, flips to refunded, restocks**, refund failure leaves order paid
- Payments service: rejects non-payable status, persists row + threads metadata on intent creation, webhook flips order + publishes succeeded event, failed-card path records error and publishes failure event, idempotent on duplicate Stripe retries, bad-signature rejected, **RefundOrder is idempotent and rejects unpaid payments**
- Notifications consumer: succeeded routes to confirmation body, failed includes reason + code, malformed JSON propagates terminal error, unknown routing key errors, mailer/load errors propagate so the consumer can retry

**Identity / catalog tests (carried over from Project 1):**
- JWT generation/validation, auth service flows (login, refresh, 2FA, password reset)
- User registration validation, category tree builder, product service
- API integration: login/refresh/logout, product CRUD, middleware enforcement
- Security: SQL injection, XSS, malformed JSON, oversized bodies, JWT tampering, user-enumeration prevention, invalid UUID, negative-value injection

### Manual testing

The React storefront at `http://localhost:8080` covers every commerce flow end-to-end. The [Commerce walkthrough](#commerce-walkthrough) above lists the click-path; the [Bundled dev services](#bundled-dev-services) section tells you where to inspect emails (Mailhog) and queue state (RabbitMQ management UI).

### Verifying encryption at rest

After placing an order, open a psql shell and inspect the raw bytes:

```bash
docker exec -it $(docker ps -qf name=db) psql -U admin -d mystore -c \
  "SELECT order_number, email_encrypted, encode(email_encrypted, 'hex') AS hex FROM orders ORDER BY created_at DESC LIMIT 1;"
```

`email_encrypted` is `bytea` ciphertext (nonce || ciphertext || GCM tag) — the hex column is what's actually on disk; plaintext never appears.

### Concurrent payment / no-overselling

With one unit of a product in stock, open two browser sessions, add the same product to each cart, and check out. The second checkout fails with `409 stock or price changed while you were checking out` because `LockProductForUpdate` (`SELECT ... FOR UPDATE`) serialises the stock decrement.

## Project Structure

```
.
├── cmd/api/main.go              # Application entrypoint, dependency wiring, graceful shutdown
├── internal/
│   ├── auth/                    # Authentication (login, refresh, 2FA, password reset)
│   ├── brand/                   # Brand management
│   ├── captcha/                 # reCAPTCHA v3 verification
│   ├── cart/                    # Shopping cart (guest + persistent, merge)
│   ├── category/                # Category tree management
│   ├── config/                  # Environment configuration loader
│   ├── crypto/                  # AES-256-GCM encryptor for order PII
│   ├── ctxkey/                  # Shared context keys (avoids import cycles)
│   ├── mailer/                  # SMTP email sender (Mailhog-compatible)
│   ├── messaging/               # RabbitMQ connection, publisher, consumer, topology
│   ├── middleware/              # Auth, admin role, optional auth, guest session
│   ├── notifications/           # Consumer that turns payment events into emails
│   ├── oauth/                   # OAuth providers (Google, Facebook)
│   ├── orders/                  # Checkout, order history, cancellation + refund
│   ├── payments/                # Stripe intents, webhook, refund client, event publisher
│   ├── product/                 # Product catalog with faceted search
│   ├── response/                # JSON response helpers
│   └── user/                    # User registration
├── migrations/                  # PostgreSQL migration files (001-021) + seed.sql
├── web/                         # React 19 + TypeScript + Vite storefront
├── static/                      # Legacy test panel (identity flows)
├── Dockerfile                   # Multi-stage build (node + golang → alpine:3.20)
├── docker-compose.yml           # Full stack (db + migrate + seed + rabbitmq + mailhog + api)
├── Makefile                     # Dev commands (up, down, reset, test, etc.)
└── .env.example                 # Environment variable template
```

## Architecture

The project follows a clean layered architecture:

```
HTTP Request → Handler (decode/validate) → Service (business logic) → Repository (database) → PostgreSQL
                                                                  ↘ Event Publisher → RabbitMQ → Notifications Consumer → Mailer → SMTP
```

Each layer communicates through Go interfaces, enabling testability with mock implementations. Import cycles (`orders ↔ payments` for the refund flow) are avoided using function-pointer adapters (`RefunderFunc`) and a shared `ctxkey` package.

The Stripe webhook handler updates the database synchronously (so the order is in `paid` before the webhook returns 200) and publishes the email side-effect to RabbitMQ. The consumer then loads the order, renders the email, and sends it via SMTP — with bounded in-process retry and a dead-letter queue for permanent failures.

### Docker Services

| Service | Image | Purpose |
|---------|-------|---------|
| `db` | postgres:16-alpine | PostgreSQL database with persistent volume + healthcheck |
| `migrate` | migrate/migrate | Runs all 21 migration files on startup; exits cleanly |
| `seed` | postgres:16-alpine | Seeds the database with test accounts, products, and reviews; exits cleanly |
| `rabbitmq` | rabbitmq:3.13-management-alpine | AMQP broker for payment events. Management UI on `15672`. `rabbitmq-diagnostics ping` healthcheck. |
| `mailhog` | mailhog/mailhog:v1.0.1 | Dev SMTP sink. SMTP on `1025`, web UI on `8025`. |
| `api` | Custom (multi-stage: node → golang → alpine) | Go API + built React storefront. Depends on `db`, `seed`, `rabbitmq` (healthy), `mailhog` (started) |

All services are orchestrated with health checks and dependency ordering. Credentials are loaded from the `.env` file via `env_file` directive — no secrets are hardcoded in the compose file. The api container overrides `DATABASE_URL`, `RABBITMQ_URL`, `SMTP_HOST`, and `SMTP_PORT` to use the docker hostnames so a fresh `.env` (copied from `.env.example`, with its host-mode localhost defaults) works unchanged.

`make up` is the only command needed to run the entire stack.
