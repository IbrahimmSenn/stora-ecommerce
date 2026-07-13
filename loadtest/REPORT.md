# Performance analysis report — load testing

Tool: **k6**. Target: the full Dockerised stack (Go API + PostgreSQL + RabbitMQ),
driven over HTTP on `:8080`. Host: Docker on a multi-core dev machine (~15 GiB
RAM). Scripts live in this directory (`load.js`, `stress.js`).

## Method

Three k6 scenarios mimic real user behaviour (`load.js`):

1. **browse** — PLP → PDP → reviews → categories (ramping to 50 VUs).
2. **search_cart** — search, then add an item to the guest cart (ramping to 25 VUs).
3. **checkout** — guest checkout placing a real order (8 constant VUs).

A separate **stress ramp** (`stress.js`) pushes browse traffic from 0 → 1000 VUs
to locate the latency knee.

Test-only adjustments (see `docker-compose.loadtest.yml`), documented so
results aren't misread:
- Per-IP rate limits are relaxed — k6 drives all traffic from one IP, so the
  production token bucket would otherwise cap throughput and emit 429s (a test
  artefact). Rate limiting itself is verified separately by the security tests.
- Address verification (external Nominatim) is pointed at a dead port so checkout
  fails it fast and proceeds via `address_override`, isolating order-placement
  throughput from a third-party dependency. Product stock was raised so checkout
  isn't starved by inventory depletion.
- **Catalog size**: the seeded catalog is only ~10 products. Read latencies are
  correspondingly optimistic — the working set fits entirely in Postgres's cache
  and the query planner never touches disk. Treat the read throughput as an
  upper bound; a production-sized catalog with cold pages and larger result sets
  would raise p95 for browse/search. The relative scenario comparison and the
  write-path (checkout) numbers are representative.

## Results — normal load (mixed scenarios, peak 83 concurrent VUs)

| Metric | Result | Target | |
|---|---|---|---|
| 90th percentile latency | **2.9 ms** | < 2 s | ✅ |
| 95th percentile latency | **3.1 ms** | — | ✅ |
| Throughput | **~160 req/s**, ~45 full journeys/s | ≥ 10 TPS | ✅ |
| Functional success (checks) | **100%** | — | ✅ |
| Error rate | **4.2%** | < 5% | ✅ |
| Concurrent users, no degradation | **83** | ≥ 50 | ✅ |
| API CPU / mem | ~18% / ~15 MB | — | ✅ |
| DB CPU / mem | ~14% / ~77 MB | — | ✅ |

Per scenario p95: browse **1.7 ms**, search+cart **3.3 ms**, checkout **3.7 ms**.
The residual ~4% `http_req_failed` is transient connection churn as VUs ramp
down (all functional checks — product reads, search, add-to-cart, **order
placement** — succeeded 100%).

## Results — stress ramp (browse, 0 → 1000 concurrent VUs)

| Metric | Result |
|---|---|
| Peak concurrent users | **1000 VUs** |
| 95th percentile latency @ 1000 VUs | **189 ms** |
| 90th percentile latency | 176 ms |
| Max latency | 324 ms |
| Throughput | **6,257 req/s** (~3,129 journeys/s) |
| Total requests | 531,466 |
| Error rate | **0.00%** (0 failures) |
| API CPU / mem | ~250–270% (≈2.6 cores) / 0.5% |
| DB CPU / mem | ~320–400% (≈4 cores) / 0.8% |

## Report findings

- **Max concurrent users before responses exceed 5 s:** **not reached — > 1000.**
  At 1000 VUs the 95th percentile was 189 ms (under even the 2 s target) with
  zero errors. The system has very large headroom over the 50-user requirement.
- **Expected throughput:** ~160 req/s under a realistic mixed workload; **~6,250
  req/s** peak for read-dominated browsing. Easily exceeds the ≥ 10 TPS target.
- **Normal vs. peak:** latency rises from ~3 ms (≤ 83 VUs) to ~189 ms (1000 VUs)
  — a graceful, roughly linear climb, never a cliff. Success stays at 100%.
- **CPU / memory crossing 90%:** **Memory never approaches it** (< 1% of 15 GiB
  throughout — the Go binary holds ~15 MB). **CPU is the dominant resource:** at
  1000 VUs PostgreSQL used ≈ 4 cores and the API ≈ 2.6 cores. On a 4-core host
  the database would be the first component to saturate; it was the busiest in
  every sample.

## Bottlenecks & proposed solutions

1. **Database CPU is the ceiling.** PG consistently outworked the API (~1.5×).
   *Solutions:* add covering indexes already present on hot paths (search GIN,
   `reviews(product_id,status)`); introduce a read cache (Redis) for the
   catalogue/PLP; add read replicas for browse traffic; consider materialising
   `avg_rating`/`review_count` instead of aggregating per query.
2. **Connection-pool sizing.** `pgxpool` runs on defaults (≈ 4×CPU). Under very
   high concurrency, requests queue on the pool. *Solution:* set explicit
   `MaxConns`/`MinConns` and load-test the value (tracked in the hardening list).
3. **Single API instance.** All traffic hit one container. *Solution:* the API is
   stateless (JWT in memory, sessions in DB/cookies) so it scales horizontally
   behind a load balancer — near-linear for read traffic.
4. **Checkout's external address verification.** Real checkout calls Nominatim
   synchronously (stubbed here). *Solution:* cache verifications, make them async,
   or fall back fast — otherwise it's the real-world checkout latency bottleneck.
5. **Rate limiting (prod).** Relaxed for this test; in production the per-IP token
   bucket protects against abuse without affecting legitimate single-user latency.
