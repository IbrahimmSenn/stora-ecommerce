# Observability

Metrics, logs, dashboards, and alerting for the platform. The goal is to see
both **technical health** (is it working?) and **business health** (is it
succeeding?) from the same stack — a system can be perfectly healthy while
carts are abandoned and revenue drops.

## Architecture

```
                     ┌───────────────────────────────────────────────┐
   ┌──────────┐      │  api  (Go)                                    │
   │ Postgres │◀─────│  • /metrics on :9091  (internal only)         │
   └────┬─────┘      │  • slog JSON logs to stdout (LOG_FORMAT=json) │
        │            │  • OTLP spans when OTEL_ENABLED=true          │
        │            └──────┬───────────────┬───────────────┬────────┘
        │ SELECT            │ scrape        │ docker logs   │ OTLP
        │ (grafana_ro)      │               │               │
        │            ┌──────▼───────┐ ┌─────▼────────┐ ┌────▼────────┐
        │            │  Prometheus  │ │   Promtail   │ │    Tempo    │
        │            │(metrics TSDB │ │ (log shipper)│ │(trace store)│
        │            │ + SLO rules) │ └─────┬────────┘ └────┬────────┘
        │            └──────┬───────┘       │ push          │
        │                   │          ┌────▼─────┐         │
        │                   │          │   Loki   │         │
        │                   │          │(log store)│        │
        │                   ▼          ▼           ▼        ▼
        │            ┌───────────────────────────────────────┐
        └───────────▶│               Grafana                 │
                     │ 5 dashboards · unified alerting ·     │
                     │ exemplars + trace_id links tie the    │
                     │ three signals together                │
                     └───────────────┬───────────────────────┘
                                     │ severity-routed notifications
                              ┌──────▼──────┐  ┌───────────────┐
                              │  Mailhog    │  │ Discord       │
                              │ (all alerts)│  │ (critical)    │
                              └─────────────┘  └───────────────┘

   Also scraped by Prometheus: cAdvisor (per-container), node-exporter (host),
   rabbitmq_prometheus (broker + queues), postgres-exporter, redis-exporter.
```

### Why this stack

| Concern | Choice | Reason |
|---|---|---|
| Metrics collection | **Prometheus client_golang** | The app exposes a pull endpoint; no agent, no code restructuring — a middleware and a handful of service-layer counters. |
| Metrics storage | **Prometheus** | Purpose-built time-series DB; the query language (PromQL) drives both dashboards and alert rules. |
| Log collection | **Promtail** | Reads container stdout via the Docker API — zero app changes beyond emitting JSON. (Promtail is superseded by Grafana Alloy upstream; Promtail 3.5 is simpler for a single host and still supported.) |
| Log storage | **Loki** | Index-light, cheap to run, label model matches Prometheus so the same mental model applies. |
| Business metrics | **Grafana → Postgres (read-only)** | Revenue, AOV, ratings are already in the database, correct and consistent. Querying them directly beats re-deriving aggregates in application counters that could drift. |
| Traces | **OpenTelemetry → Tempo** | One request produces one trace spanning HTTP → SQL → AMQP publish → consumer. Exemplars and `trace_id` in logs make latency spikes clickable down to the exact slow query. Opt-in (`OTEL_ENABLED=true`) so the default stack stays lean. |
| Dependency metrics | **rabbitmq_prometheus, postgres-exporter, redis-exporter** | The broker and database are where incidents start; queue/DLQ depth and server-side DB stats can't be derived from app-side counters. |
| Visualization + alerting | **Grafana** | One tool for Prometheus, Loki, Tempo, and Postgres; provisioned dashboards + unified alerting in the repo. |

Everything is pinned by image tag and provisioned from `observability/` — the
stack is reproducible with one command and versioned in git.

### Meaningful metrics, not vanity metrics

Every metric here answers a decision: *revenue by category* → where to promote;
*guest-vs-registered AOV* → whether to push account creation; *p95 by route* →
which endpoint to optimise; *failed logins vs 429s* → whether the rate limiter
is sized right. Raw totals with no decision attached (e.g. "total page views")
are deliberately absent. High-cardinality labels (user IDs, session tokens, raw
URLs, raw Stripe error strings) are never used — route labels come from the Chi
**pattern** (`/api/v1/products/{id}`), and payment/checkout reasons map to fixed
enums.

## Setup

The monitoring stack is a compose **profile**, off by default.

```bash
make up                       # app + database (in one terminal, or detached)
make monitoring-up            # docker compose --profile monitoring up -d
make seed-history             # ~60 days of users/orders/reviews for the business dashboards
make loadtest                 # baseline browse/search/checkout traffic (technical panels)
make hostile                  # failed logins, rate-limit bursts, forged webhooks (security panels)
```

| Service | URL | Notes |
|---|---|---|
| Grafana | http://localhost:3001 | `admin` / `admin` (or `GRAFANA_ADMIN_PASSWORD`) |
| Prometheus | http://localhost:9090 | Targets at /targets, SLO rules at /rules |
| Mailhog | http://localhost:8025 | Alert emails land here |
| API metrics | `api:9091/metrics` | Internal to the compose network — not published to the host |
| Tempo | `tempo:3200` | Trace store, internal; query it through Grafana Explore |
| Exporters | `rabbitmq:15692`, `postgres-exporter:9187`, `redis-exporter:9121` | Internal; visible on Prometheus /targets |

Tracing is opt-in: set `OTEL_ENABLED=true` in `.env` and restart the api (the
monitoring profile must be up so Tempo exists; export failures are non-fatal
either way). The five dashboards are under Grafana → Dashboards →
**I Love Shopping**.

## Dashboards

Each dashboard opens with stat tiles (headline numbers) and drills into
time-series below. Business dashboards default to a 30-day window; technical and
security default to 1 hour. Money is shown in currency units (converted from the
integer-cent storage), latency in seconds, rates in req/s.

### 1. Business Intelligence *(Postgres)*
Financial health of the platform.

| Metric | Why it matters |
|---|---|
| Daily revenue | The top-line trend; the number the business is ultimately judged on. |
| Average order value | Revenue quality — rising AOV means more per transaction, not just more transactions. |
| Cart abandonment rate | Carts with items, idle >1h, with no subsequent order ÷ carts with items. Direct funnel-leak signal. |
| Revenue by category | Where the money comes from; flags categories to promote or delist. |
| Registrations (daily) | Top-of-funnel growth. |
| **Correlated: AOV guest vs registered** | See *Correlated insights* below. |

### 2. Product & Customer *(Postgres)*
Product performance and customer behaviour.

| Metric | Why it matters |
|---|---|
| Top 10 products (revenue + units) | What to keep in stock and feature. |
| Average rating by category | Quality signal per category; low ratings predict returns and churn. |
| Orders by customer type (daily) | Guest vs registered volume — conversion opportunity sizing. |
| Review volume (daily) | Engagement + how much rating data is actually trustworthy. |
| Conversion funnel + drop-off | Views → add-to-cart → checkout → paid (from `user_activity` + `orders`). The biggest drop names the next UX investment. |
| View → paid conversion (daily) | A dip with steady traffic means the shop is losing buyers, not visitors. |
| **Correlated: rating vs revenue by category** | See below. |

### 3. Technical Performance *(Prometheus + Loki)*
System health, latency, reliability.

| Metric | Why it matters |
|---|---|
| Request rate, 5xx ratio, p95 | The RED method — the three numbers that describe service health. |
| p95 latency by endpoint | Which route is slow (search / cart / checkout / auth are called out). |
| HTTP errors by status | 4xx (client) vs 5xx (us) split. |
| Request volume by route | Usage patterns + capacity planning. |
| DB query p95 by operation + pool gauges | The database is the usual bottleneck under load; this is the early warning. |
| Payment failures by reason | Checkout/gateway reliability, broken out by Stripe error class. |
| Host CPU / memory + API RSS | The "90% load" line for the performance report (see caveat below). |
| Recent API errors (Loki) | Jump straight from a metric spike to the stack traces. |
| RabbitMQ queue/DLQ depth, throughput, consumers | A growing queue = the email consumer is behind; anything in `payments.emails.dlq` needs a human. |
| Postgres server: tuples, connections, cache hit | Server-side view that complements the client-side pgx pool panels — disagreement between them points at a non-api connection eater. |
| Web Vitals p75 (LCP/INP/FCP/TTFB, CLS) | Real-user experience from the browser, not synthetic — the numbers shoppers actually feel. |

### 4. Security *(Prometheus + Loki)*
Authentication anomalies and threat indicators.

| Metric | Why it matters |
|---|---|
| Failed logins by reason | Brute-force / credential-stuffing signal (`invalid_credentials` vs `invalid_2fa`). |
| Login success ratio | A collapsing ratio = an attack in progress. |
| Token refreshes by outcome | Failures include replayed (already-used) tokens — the rotation-theft signal. |
| Password reset events | Spikes can indicate account-takeover attempts. |
| Rate-limit rejections by limiter | How much abuse the token buckets are absorbing. |
| **Correlated: login failures vs auth 429s** | See below. |
| Security events (Loki) | The structured log lines behind the counters, with client IP. |

### 5. SLO / Error Budget *(Prometheus)*
Reliability as a budget, not a mood. See [SLOs and error budgets](#slos-and-error-budgets).

| Metric | Why it matters |
|---|---|
| Availability + latency SLIs (5m/1h) | The two user-facing promises: requests succeed, and they're fast. |
| Error budget remaining (30d) | How much unreliability the month can still absorb — the number that decides "ship it" vs "stabilise first". |
| Burn rates (5m/30m/1h/6h) | How fast the budget is being spent; the alert thresholds are drawn on the panel. |

## Correlated insights

A single metric is often ambiguous; these panels connect two signals to reveal
something neither shows alone.

1. **AOV — guest vs registered** (Business Intelligence). Registered customers
   spend materially more per order than guests. The overall AOV number hides
   this. The gap quantifies the value of converting guests to accounts, which is
   a concrete lever (checkout account-creation nudges).

2. **Rating vs revenue by category** (Product & Customer). Ratings and revenue
   disagree in both directions. *High rating + low revenue* = an under-marketed
   product line worth promoting. *High revenue + mediocre rating* = today's
   income sitting on tomorrow's churn risk. Sorting by revenue alone, or by
   rating alone, would miss both.

3. **Login failures vs auth-limiter 429s** (Security). Both rising together =
   an attack that the rate limiter is successfully throttling (the defence
   works). Failures rising while 429s stay flat = the attacker is operating
   *under* the limiter threshold — tighten `AUTH_RATE_LIMIT_RPS/BURST`. Either
   series alone can't distinguish "under attack but defended" from "under attack
   and exposed".

## Alerting

Grafana-managed rules (provisioned in
`observability/grafana/provisioning/alerting/`), evaluated every minute against
Prometheus. Notifications route by severity:

| Severity | Contact point | Channels |
|---|---|---|
| critical | `ops-critical` | email → Mailhog **and** Discord webhook |
| warning (default) | `ops-email` | email → Mailhog (http://localhost:8025, offline) |

To get real Discord messages, set `GRAFANA_DISCORD_WEBHOOK` in `.env` and
restart grafana. Without it, compose substitutes a placeholder webhook URL so
provisioning stays valid — the Discord delivery fails visibly in Grafana's
notification history while email keeps working.

| Alert | Condition | For | Severity |
|---|---|---|---|
| **APIDown** | `up{job="api"} == 0` | 1m | critical |
| **HighErrorRate** | 5xx ratio > 5% | 2m | critical |
| **AuthFailureSpike** | > 25 failed logins / 5m | 2m | warning |
| **HighLatencyP95** | p95 latency > 1.5s | 3m | warning |
| **SLOAvailabilityFastBurn** | availability burn > 14.4× on 5m **and** 1h | 2m | critical |
| **SLOAvailabilitySlowBurn** | availability burn > 6× on 30m **and** 6h | 5m | warning |
| **SLOLatencyFastBurn** | latency burn > 14.4× on 5m **and** 1h | 2m | critical |
| **SLOLatencySlowBurn** | latency burn > 6× on 30m **and** 6h | 5m | warning |
| **PaymentFailureRatio** | payment success < 90% / 15m (min 5 attempts) | 5m | warning |

The static rules (HighErrorRate, HighLatencyP95) and the SLO burn rules watch
the same failure modes at different rigor: the static pair is simple to reason
about in a demo, the burn pair is how the thresholds should really be derived
— from a promise to the user rather than a round number. Both are provisioned;
in production the burn pair would replace the static pair.

Every rule carries a `runbook_url` annotation pointing at the firing drill
below, so the notification itself tells the responder what to do next.

**Cause/symptom suppression.** When the API is down, the error-rate and latency
rules would fire too — three pages for one problem. Grafana-managed alerting
has no Alertmanager-style `inhibit_rules`, so the suppression is done in the
query: HighErrorRate and HighLatencyP95 are guarded with
`and on() (up{job="api"} == 1)`, which empties their result during an outage,
and `noDataState: OK` keeps them green. Only APIDown pages; the symptom rules
come back automatically once the target is up.

### Actionable alerts, and avoiding alert fatigue

Not every metric is an alert — dashboards are for exploration, alerts are for
things that need a human **now**. The four rules were chosen because each maps
to a clear action (restart the service, find the failing route, check for an
attack, look at DB saturation), and every alert annotation names that next step.

Fatigue controls live in the notification policy (`policies.yml`):

- **`for` durations** — a rule must breach for 1–3 minutes before firing, so a
  transient blip during a deploy doesn't page anyone.
- **Grouping by `alertname`** — one notification per problem, not one per series.
- **`group_wait` 30s / `group_interval` 5m** — related alerts are batched.
- **`repeat_interval` 4h** — an unresolved alert re-notifies every 4 hours, not
  every evaluation.
- **Severity routing** — `critical` escalates to email + Discord, `warning`
  stays on email only, so the noisy channel is reserved for page-worthy events.
- **Cause/symptom suppression** — symptom rules are query-guarded on
  `up{job="api"} == 1` so an API outage produces one page, not three.
- **Thresholds set above normal** — "normal" was read off the dashboards first
  (5xx ~0%, p95 well under 0.5s, near-zero failed logins at rest), then
  thresholds set with headroom so only genuine anomalies trip them.

### Firing each alert on demand (review runbook)

| Alert | How to trigger | Recovery |
|---|---|---|
| **APIDown** | `docker compose stop api` → wait ~90s | `docker compose start api` (sends a RESOLVED email) |
| **HighErrorRate** | `docker compose stop db` while `make loadtest` runs (DB-backed routes 500) | `docker compose start db` |
| **AuthFailureSpike** | `make hostile` — the failed-login scenario drips past the auth limiter and accumulates > 25 in 5m | ends with the run |
| **HighLatencyP95** | `make loadtest` at high VU counts, or stress the DB | ends with the run |
| **SLOAvailabilityFastBurn** | same drill as HighErrorRate, held for ~6–7 min — watch the burn-rate panels on the SLO dashboard cross 14.4 | `docker compose start db` |
| **PaymentFailureRatio** | `make hostile` — forged Stripe webhooks make 100% of payment attempts fail; fires once ≥ 5 attempts accumulate in 15m | ends with the run |

After each, check http://localhost:8025 for the email (critical alerts also
hit Discord if the webhook is set) and the alert state in Grafana → Alerting.
Note the suppression while APIDown is firing: HighErrorRate and the burn
alerts stay green because their queries are guarded on `up == 1` — that's the
dedup working, not a gap.

## SLOs and error budgets

Two Service Level Objectives are defined over the HTTP request stream, with
recording rules in `observability/prometheus/rules.yml`:

| SLO | Target | Error budget (30d) |
|---|---|---|
| Availability | 99.5% of requests answer without a 5xx | 0.5% of requests |
| Latency | 95% of requests answer under 500ms | 5% of requests |

**Burn rate** is how fast the budget is being spent: 1× means spending exactly
the monthly budget, 14.4× means the whole month's budget would be gone in ~2
days. The alerts are **multi-window**: a long window (1h/6h) proves the burn is
sustained, a short window (5m/30m) proves it's still happening — so a resolved
incident stops paging within minutes instead of dragging the long window
around. Fast burn (14.4× on 5m+1h) pages as critical; slow burn (6× on
30m+6h) warns. These are the canonical Google SRE numbers, and they genuinely
fire in a demo: a full outage under load crosses fast burn in ~6 minutes.

Why this beats static thresholds: "p95 > 1.5s" is a number picked by feel;
"we promised 95% of requests under 500ms and we're eating the annual budget
40× too fast" is a promise to the user with the math attached. The error
budget also gives the honest answer to "can we ship on Friday?" — yes if
budget remains, no if it's spent.

The **PaymentFailureRatio** alert is the business-symptom counterpart: it
watches the customer-visible outcome (payments failing) regardless of cause.
A "zero paid orders while traffic is high" alert was considered and rejected
for this environment — the k6 scenarios never complete a Stripe payment, so
it would fire on every load test; in production it would be the next alert
to add.

The 30-day budget gauges read low at first: recording rules only exist from
the moment they shipped, so `avg_over_time(...[30d])` sees a short history
until data accumulates.

## Distributed tracing

With `OTEL_ENABLED=true`, every request produces one trace: the HTTP server
span (named by Chi route pattern, e.g. `GET /api/v1/products/{id}`), child
spans for each SQL query (via a pgx tracer composed with the Prometheus one —
pgx only accepts a single tracer), and, for checkout/payment flows, AMQP
producer and consumer spans linked across the broker by W3C trace context in
the message headers. Sampling is parent-based, ratio via `OTEL_SAMPLE_RATIO`
(default 1.0 — demo-friendly; production would sample down).

The three signals are cross-linked in Grafana:

- **Metrics → traces**: the latency histogram records the trace ID as an
  **exemplar** on each sample — the dots on the "API latency p95 by endpoint"
  panel click through to the exact slow trace in Tempo.
- **Logs → traces**: access logs carry `trace_id`/`span_id`; a Loki derived
  field turns them into a Tempo link.
- **Traces → logs**: the Tempo datasource is configured to jump back to the
  Loki lines for a trace.

That triangle is the debugging workflow: see a p95 spike → click an exemplar →
read the span tree to find the slow query → jump to the logs around it.
Everything no-ops when `OTEL_ENABLED` is unset — the default stack runs no
OTel SDK, and the instrumentation points cost nothing.

## Data generation

Two components feed the dashboards — historical records for the business panels,
live traffic for the technical/security panels.

### Seed data (`make seed-history`)
`cmd/seedhistory` writes ~60 days of backdated data directly via SQL (the normal
repositories can't backdate `created_at`): **~300 users**, **~900 orders**
(70% registered / 30% guest, realistic status mix), **~150 abandoned carts**,
**~500 reviews**, and **~10k funnel activity events** (the views and
add-to-carts that led to each seeded order, plus ~800 browse-only sessions
that never convert — the honest top of the funnel), all with timestamps spread
across the window. It reuses the
app's `crypto.Encryptor` so seeded emails are encrypted exactly like real ones.
It is deterministic and idempotent (fixed RNG seed, fixed UUIDs,
`ON CONFLICT DO NOTHING`) — safe to re-run after `make reset`. Two patterns are
engineered on purpose so the correlated panels have signal: registered orders
carry more items than guest orders (the AOV gap), and the highest-revenue
categories are given mediocre ratings while a quiet category rates highly (the
rating/revenue divergence).

### Traffic simulation (`make hostile`, `make loadtest`)
- `loadtest/load.js` (existing) — browse / search / checkout, drives request
  volume, latency, and DB metrics.
- `loadtest/hostile.js` (new) — four hostile scenarios: failed logins (wrong
  passwords), rate-limit bursts against catalog + login, refresh-token churn
  (garbage tokens), and forged-signature Stripe webhooks + malformed checkouts.
  Drives every security metric and the payment-failure panel. **Run against the
  default stack** (`make up`), not the load-test override — that override raises
  the rate limits so high that nothing would be throttled.

## Data protection

The Grafana Postgres datasource connects as **`grafana_ro`** (migration
`000037`): `SELECT` on commerce tables only, and **column-level** grants on
`users` (`id, role, created_at`) and `user_activity` (everything except
`search_query`, which is free-text user input — migration `000039`) so the
encrypted PII columns (`email_encrypted`, `email_hmac`, `password_hash`) and
raw search strings are unreadable even if the Grafana credentials leak.

`postgres-exporter` uses a **separate `postgres_exporter` role** (migration
`000038`, `pg_monitor`) rather than widening `grafana_ro` — `pg_monitor`
exposes `pg_stat_activity` query text, which can contain literal parameter
values, so it gets its own credential and blast radius.

The API `/metrics` endpoint listens on a **separate internal port (9091)**
that is never published to the host — only Prometheus, inside the compose
network, can reach it. The Web Vitals ingest endpoint (`POST /api/v1/vitals`)
is public by necessity but stores nothing user-identifying: whitelisted metric
names, bounded values, 1 KB body cap, general rate limiter.

## Known limitation — cAdvisor on Docker Desktop / WSL2

`cadvisor` and `node-exporter` are included for container- and host-level
resource metrics. **node-exporter works everywhere** (host CPU, memory, load —
this is what the Technical dashboard's resource panels use). **cAdvisor cannot
map cgroups to individual containers under Docker Desktop / WSL2** — containers
run inside Docker Desktop's VM with a cgroup layout cAdvisor can't attribute, so
its per-container series carry no compose-service label. On a **native Linux
host** cAdvisor works normally and you can add per-container CPU/memory series
(`container_cpu_usage_seconds_total` by `container_label_com_docker_compose_service`)
to the resource panels. Until then, host metrics (node-exporter) plus the API's
own process metrics (Go/process collectors on `/metrics`) cover resource usage.
