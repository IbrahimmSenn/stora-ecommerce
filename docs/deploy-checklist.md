# Deploying the public demo

Step-by-step guide for putting the shop on a real URL as a public portfolio
demo: Hetzner VPS + Docker Compose + Caddy (automatic Let's Encrypt TLS) +
Stripe test mode. Budget ≈ €4–5/month (VPS + domain).

The app itself needs no code changes for this — everything below is accounts,
DNS, and a server `.env`.

## 0. Rotate the leaked secrets (before the repo goes public)

Commit `d9b0751` leaked the Google OAuth client secret and reCAPTCHA keys into
git history (documented in `.gitleaksignore`). They cannot be un-leaked;
**rotate them**:

- [ ] Google Cloud Console → APIs & Services → Credentials → your OAuth 2.0
      client → **add a new client secret, delete the old one**.
- [ ] reCAPTCHA admin console (https://www.google.com/recaptcha/admin) →
      create a **new site key/secret pair**; stop using the old pair.

Only after rotation is it safe to publish the repository.

## 1. Domain

- [ ] Buy a domain (~€10–12/yr — Porkbun, Namecheap, …).
- [ ] After step 2, add a DNS **A record** for `@` (and `www` if you want it)
      pointing at the server's IPv4. If you serve `www` too, add it to
      `deploy/Caddyfile` (`{$DOMAIN}, www.{$DOMAIN} { ... }`).

## 2. Server (Hetzner CX22 or similar — 2 vCPU, 4 GB, x86)

- [ ] Create the server with Ubuntu 24.04 and your SSH key.
- [ ] Basic hardening:
  ```bash
  adduser deploy && usermod -aG sudo deploy
  # copy your SSH key to deploy, then disable root + password login in /etc/ssh/sshd_config
  ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable
  ```
- [ ] Install Docker Engine + Compose plugin from docker.com (NOT Ubuntu's
      docker.io — the `!override` tag in docker-compose.prod.yml needs
      Compose ≥ 2.24.4). Verify: `docker compose version`.
- [ ] `usermod -aG docker deploy`

## 3. App on the server

- [ ] ```bash
      sudo mkdir -p /opt/stora && sudo chown deploy /opt/stora
      git clone https://github.com/IbrahimmSenn/stora-ecommerce.git /opt/stora
      cd /opt/stora && cp .env.example .env
      ```
- [ ] Fill `/opt/stora/.env` (never commit it):
  ```bash
  APP_ENV=production
  DEMO_MODE=true
  DOMAIN=<your-domain>
  BASE_URL=https://<your-domain>
  CORS_ORIGINS=https://<your-domain>
  COMPOSE_FILE=docker-compose.yml:docker-compose.deploy.yml:docker-compose.prod.yml
  SMOKE_URL=https://<your-domain>

  POSTGRES_PASSWORD=<openssl rand -hex 16>
  DATABASE_URL=postgres://admin:<same-password>@db:5432/mystore?sslmode=disable
  JWT_SECRET=<openssl rand -hex 32>
  ENCRYPTION_KEY=<openssl rand -hex 32>
  ADMIN_PASSWORD=<strong password, min 12 chars — your admin login>
  RABBITMQ_USER=app
  RABBITMQ_PASSWORD=<openssl rand -hex 16>
  RABBITMQ_URL=amqp://app:<same-password>@rabbitmq:5672/

  SKIP_CAPTCHA=false
  RECAPTCHA_SITE_KEY=<rotated key>
  RECAPTCHA_SECRET_KEY=<rotated secret>
  GOOGLE_CLIENT_ID=<id>
  GOOGLE_CLIENT_SECRET=<rotated secret>

  STRIPE_SECRET_KEY=sk_test_...
  STRIPE_PUBLISHABLE_KEY=pk_test_...
  STRIPE_WEBHOOK_SECRET=whsec_...   # from step 4, not from `stripe listen`

  SMTP_HOST=smtp-relay.brevo.com    # step 5; leave empty to disable email
  SMTP_PORT=587
  SMTP_USER=<brevo login>
  SMTP_PASS=<brevo smtp key>
  SMTP_FROM=<verified sender>
  ```
  Notes: the encryption key encrypts PII at rest — losing it orphans that
  data. `DATABASE_URL` keeps `sslmode=disable` because Postgres only exists on
  the private compose network (demo mode allows this; the default `secret`
  password is still rejected).

## 4. Stripe webhook (test mode)

`stripe listen` is dev-only. For the server:

- [ ] Stripe Dashboard (test mode) → Developers → Webhooks → **Add endpoint**:
      `https://<your-domain>/api/v1/webhooks/stripe`, events
      `payment_intent.succeeded` + `payment_intent.payment_failed`.
- [ ] Copy the endpoint's `whsec_...` into `STRIPE_WEBHOOK_SECRET` in `.env`.

## 5. External services for the new domain

- [ ] **Brevo** (free, 300 emails/day): sign up, verify a sender address,
      SMTP & API → SMTP keys → fill the `SMTP_*` vars. Skip to run the demo
      without emails (sending becomes a logged no-op).
- [ ] **reCAPTCHA console**: add `<your-domain>` to the site key's domain list.
- [ ] **Google OAuth console**: add
      `https://<your-domain>/api/v1/auth/oauth/google/callback`
      to Authorized redirect URIs.

## 6. First deploy (manual)

- [ ] Make the GHCR package public: GitHub → repo → Packages → package
      settings → Change visibility. (Or `docker login ghcr.io` on the server
      with a read-only PAT.)
- [ ] ```bash
      cd /opt/stora
      scripts/deploy.sh ghcr.io/ibrahimmsenn/stora-ecommerce:latest
      ```
      This pulls the image, runs migrations + seed, starts everything
      (including Caddy, which fetches the Let's Encrypt cert on first hit),
      and smoke-tests against `SMOKE_URL`.
- [ ] Check it: open `https://<your-domain>` — demo banner visible, catalog
      loads, checkout works with `4242 4242 4242 4242`, webhook deliveries
      show 200 in the Stripe dashboard, admin login works with
      `ADMIN_PASSWORD` (and `admin123` does NOT).

## 7. Continuous deployment

- [ ] Create a dedicated deploy keypair: `ssh-keygen -t ed25519 -f deploy_key`;
      append `deploy_key.pub` to `~deploy/.ssh/authorized_keys` on the server.
- [ ] GitHub repo → Settings → Secrets and variables → Actions:
      `DEPLOY_HOST` (server IP), `DEPLOY_USER` (`deploy`),
      `DEPLOY_SSH_KEY` (private key contents).
- [ ] Merge to `main` → the pipeline's `deploy` job goes live automatically.
      Until the secrets exist the job skips gracefully.

Rollback at any time on the server:
`scripts/rollback.sh ghcr.io/ibrahimmsenn/stora-ecommerce:<previous-sha> [backups/db-....sql]`

## 8. Optional: monitoring on the server

```bash
docker compose --profile monitoring up -d
```
Grafana/Prometheus bind to 127.0.0.1 in the prod overlay — reach them via an
SSH tunnel: `ssh -L 3001:localhost:3001 deploy@<server>` then open
http://localhost:3001. Set `GRAFANA_ADMIN_PASSWORD` in `.env` first.
On 4 GB RAM the full stack fits; watch memory if you enable everything.

## Verification checklist (after any deploy)

- [ ] `scripts/smoke.sh https://<your-domain>` passes.
- [ ] Full checkout with `4242 4242 4242 4242` → order flips to paid,
      confirmation email arrives (if SMTP configured).
- [ ] Decline card `4000 0000 0000 9995` → clean failure message.
- [ ] Google OAuth round-trip works.
- [ ] `admin123` does not log in as admin; `ADMIN_PASSWORD` does.
- [ ] From outside, only ports 22/80/443 answer
      (`nmap <server-ip>` or `nc -zv <server-ip> 5432 5672 8080` all refused).
- [ ] Response headers include `Strict-Transport-Security`; cookies are `Secure`.
