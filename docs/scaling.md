# Scaling & microservices-readiness

This is a modular monolith: one deployable, but organized so it can be split
into services without a rewrite. This document records the seams, the order to
extract them, and what has to move to shared infrastructure first.

## Current shape

Each domain lives in its own package under `internal/` with a strict three-layer
split behind interfaces:

- **handler** — HTTP only (parse, call service, write response).
- **service** — business logic, no SQL, no HTTP.
- **repository** — SQL only, returns domain errors.

Packages: `auth`, `user`, `product`, `category`, `brand`, `cart`, `orders`,
`payments`, `review`, `address`, `recommend`, `contact`, plus cross-cutting
`middleware`, `audit`, `cache`, `messaging`, `crypto`. Because every dependency
is an interface, a package can be lifted into its own process by swapping the
in-process call for an HTTP/gRPC client or an event — callers don't change.

## What already decouples

- **RabbitMQ** decouples payments from orders today: the payment flow publishes
  status to an exchange (with a dead-letter queue) and the order service
  consumes it. That is a real service boundary already in production form.
- **Stateless request path.** Auth is JWT (access token in memory, refresh in an
  HttpOnly cookie) — no server-side session affinity. The only per-instance
  state was the rate limiter; see below.

## Shared state — done

For more than one API instance behind a load balancer, per-instance state must
move to a shared store. `REDIS_URL` switches these from in-memory to Redis:

- **Rate limiting** (`internal/middleware`): a `limiterStore` interface with an
  in-memory token bucket (default) and a Redis token bucket (atomic Lua script)
  that enforces one global budget per client IP across all instances. Fails open
  on a Redis error so an outage never takes the site down.
- **Read cache** (`internal/cache`): a `Cache` interface (in-memory TTL default,
  Redis when configured) used for the category list/tree. Add the same wrapper
  to brands or other rarely-changing reads as needed.

Start the bundled Redis with `docker compose --profile scale up` and set
`REDIS_URL=redis://redis:6379/0`.

## Extraction order (when traffic warrants it)

1. **Payments** first — already event-driven via RabbitMQ; mostly a deployment
   split, minimal code change.
2. **Catalog read path** (`product`, `category`, `brand`, `review`) — read-heavy,
   cacheable, benefits most from independent scaling and read replicas. Ratings
   are already denormalized onto `products` (trigger-maintained) so the listing
   query does no aggregate join.
3. **Orders / cart** — transactional; split last and keep on its own schema.

Each service gets its own schema (or database) so they can scale and be deployed
independently; cross-service reads become API calls or events, never shared
tables.

## Database scaling levers (in rough order)

- Connection pool is sized explicitly (`DB_MAX_CONNS`, etc.); budget it against
  Postgres `max_connections` across all replicas — add **pgbouncer** before the
  replica count outgrows the connection budget.
- **Read replicas** for the catalog read path; route reads to replicas, writes to
  primary.
- Indexes are in place incl. a GIN full-text index for search and a `pg_trgm`
  index for autocomplete; revisit with `EXPLAIN` as data grows.
- Product images are pre-rendered into size variants and served as `.webp` under
  `/media` — front this with a CDN and the app stops serving image bytes.

## CIA triad mapping

See [security.md](security.md) for the full data-protection model. In scaling
terms: **Availability** is the focus here — horizontal scale-out, fail-open rate
limiting, read replicas, and the message queue's retry/dead-letter handling all
keep the system serving under load and partial failure, without weakening the
**Confidentiality** (encryption at rest, TLS) or **Integrity** (ACID
transactions, inventory locking, audit log) guarantees described there.
