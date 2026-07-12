# CI/CD Pipeline

Single pipeline in [`.github/workflows/pipeline.yml`](../.github/workflows/pipeline.yml),
triggered by every push to `main` (and the `integrator` working branch) plus pull
requests into `main`. Five stages run in sequence; any failure halts the pipeline
before the next stage.

```
┌──────────────────────┐   ┌───────────────────┐   ┌───────────────┐   ┌──────────────────┐   ┌─────────────────┐
│ 1 Build & Test       │   │ 2 Security Scan   │   │ 3 Migrations  │   │ 4 Core Delivery  │   │ 5 Deploy (main) │
│ backend-test         │──▶│ security-code     │──▶│ up from zero  │──▶│ deploy ephemeral │──▶│ ssh to server   │
│ frontend-test        │   │ security-secrets  │   │ down 1 / up 1 │   │ smoke tests      │   │ deploy.sh image │
│ docker-build (+test) │   │ security-image    │   │ seed applies  │   │ publish to GHCR  │   │ smoke on live   │
└──────────────────────┘   └───────────────────┘   └───────────────┘   └──────────────────┘   └─────────────────┘
```

## Stage 1 — Build & Test

| Job | What it does |
|---|---|
| `backend-test` | `go vet` + the full Go suite (~220 tests) via `gotestsum`; JUnit report uploaded as the `backend-test-report` artifact. |
| `frontend-test` | `npm ci`, ESLint (errors fail the job), Vitest with JUnit report, then the type-checked production build (`tsc -b && vite build`). |
| `docker-build` | Builds `Dockerfile --target test` (the Go suite runs **inside the container**), then the runtime image. The image is saved as the `docker-image` artifact so later stages scan and deploy the exact bytes that were tested. |

## Stage 2 — Security & Dependency Scan

| Job | Tool | Gate |
|---|---|---|
| `security-code` | **gosec** (SAST: SQLi, XSS, path traversal, weak crypto, …) | Fails on HIGH severity at MEDIUM+ confidence. Full JSON report (all severities) uploaded as `security-reports`. |
| `security-code` | **govulncheck** | Fails on any vulnerability *reachable from our code* — call-graph analysis, so unreachable CVEs in transitive deps don't block. |
| `security-code` | **npm audit** | Fails on high/critical advisories in frontend dependencies. |
| `security-secrets` | **gitleaks** | Scans the working tree **and the entire git history** (`fetch-depth: 0`) for API keys, tokens, private keys. Any finding fails the job. |
| `security-image` | **trivy** | Scans the built container (OS packages + Go binary). Fails on fixable HIGH/CRITICAL. |

### Risk assessment / accepted findings

- **False positives** are annotated in-code with `#nosec <rule>` plus a
  justification comment (e.g. the Nominatim client is flagged as SSRF, but the
  host comes from operator config and user input is query-encoded only).
- **Doc/test fixtures** (README example JWTs, dummy AES keys in unit tests) are
  allow-listed by regex in [`.gitleaks.toml`](../.gitleaks.toml).
- **Historical leak**: commit `d9b0751` briefly committed the Google OAuth
  client secret and reCAPTCHA keys in `docker-compose.yml`. Remediation is
  **rotation** (git history is immutable once shared); the three findings are
  pinned by fingerprint in [`.gitleaksignore`](../.gitleaksignore) so new leaks
  still fail the build.
- gosec MEDIUM/LOW findings (cookie flags governed by `APP_ENV`, 0755 dirs,
  log-injection taint warnings) stay visible in the JSON report artifact but do
  not block; each is deliberate behaviour documented in `docs/security.md`.

## Stage 3 — Database Migration

Against a disposable Postgres 16 service container, using the same
`migrate/migrate:v4.17.0` image the compose stack uses:

1. **Apply all migrations from an empty database** — proves ordering and syntax
   (golang-migrate runs each file in a transaction; a failure leaves the
   version marked dirty rather than half-applied).
2. **`down 1` then `up 1`** — proves the newest migration is actually
   reversible, not just that a `.down.sql` file exists.
3. **`seed.sql` applies cleanly** on the migrated schema (it is idempotent —
   `ON CONFLICT DO NOTHING` throughout — so re-deploys are safe).

On a real deploy, `scripts/deploy.sh` takes a `pg_dump` backup **before**
migrations run (see Stage 4).

## Stage 4 — Core Delivery

Runs on pushes only (not PRs). The job:

1. Loads the tested image and writes an `.env` — `JWT_SECRET`/`ENCRYPTION_KEY`
   are generated per run with `openssl rand`; Stripe keys come from repository
   secrets when configured, with harmless placeholders otherwise.
2. Runs [`scripts/deploy.sh`](../scripts/deploy.sh): database backup (when one
   exists) → migrations → `docker compose up --wait` → post-deploy validation.
3. Validation is [`scripts/smoke.sh`](../scripts/smoke.sh): `/ready`
   (API + Postgres + RabbitMQ health), product listing, search, demo-user
   login, add-to-cart, sitemap — the critical user flows.
4. On success the image is pushed to GHCR as
   `ghcr.io/<owner>/<repo>:<commit-sha>` (every green build is a rollback
   target) and additionally `:latest` on `main`.
5. On failure the job dumps service logs and tears the environment down; the
   image is **not** published.

## Stage 5 — Production Deploy

Runs only on pushes to `main`, after delivery publishes the image. The job
SSHes to the server (`DEPLOY_HOST`/`DEPLOY_USER`/`DEPLOY_SSH_KEY` repository
secrets), pulls the repo, and runs the same
[`scripts/deploy.sh`](../scripts/deploy.sh) with the GHCR image for the exact
commit that passed all previous stages. The server's own `.env` selects the
production compose overlay (`COMPOSE_FILE=...:docker-compose.prod.yml`) and
points the post-deploy smoke tests at the public URL (`SMOKE_URL`). While the
secrets are unset the job skips gracefully, so the pipeline stays green on
forks and before a server exists. Full server setup:
[deploy-checklist.md](deploy-checklist.md).

### Deploying to any other host

The delivery scripts are host-agnostic — anything with Docker Compose:

```sh
cp .env.example .env    # fill in real values (once per host)
scripts/deploy.sh ghcr.io/<owner>/<repo>:<sha>
```

### Rollback

```sh
# application only (newer migrations are backward-compatible):
scripts/rollback.sh ghcr.io/<owner>/<repo>:<previous-sha>

# application + database, restoring the pre-migration pg_dump taken by deploy.sh:
scripts/rollback.sh ghcr.io/<owner>/<repo>:<previous-sha> backups/db-<timestamp>.sql
```

Both paths finish by re-running the smoke tests to confirm the rolled-back
version works. For reversible schema changes prefer
`make migrate-down` over a dump restore — it preserves data written after the
deploy.

## Demo script (review walkthrough)

1. **Stable start** — `main` is green, app deployed via `scripts/deploy.sh`.
2. **Failing test** — break an assertion in any `*_test.go`, push: Stage 1
   fails, the JUnit artifact names the failing test, nothing downstream runs.
3. **Security failure** — add a fake `AWS_SECRET_ACCESS_KEY=...` or an
   `sk_live_...` string to any tracked file, push: `security-secrets` fails and
   halts the pipeline before migration/delivery.
4. **Happy path** — revert both, add a schema change (e.g. a new index
   migration `000037_*.up.sql`/`.down.sql`), push: all stages pass, the
   migration is validated (up from zero + down/up), the image is published.
5. **Rollback** — `scripts/rollback.sh <previous-image> [backup.sql]`, then
   `scripts/smoke.sh` confirms the previous version serves traffic and the
   database is consistent.
