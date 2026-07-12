// k6 hostile-traffic simulation — drives the Security dashboard and the
// AuthFailureSpike alert with realistic bad behaviour:
//   1. failed_logins       — wrong passwords at ~1/s (credential stuffing)
//   2. rate_limit_burst    — bursts against catalog + login (429s on both limiters)
//   3. refresh_churn       — garbage refresh tokens between valid rotations
//   4. invalid_payments    — unsigned Stripe webhooks + malformed checkouts
//
// Run against the DEFAULT stack (make up), NOT the loadtest override — that
// override raises the rate limits to a million and nothing would throttle:
//   docker run --rm --network host -v "$PWD/loadtest:/s" grafana/k6 run /s/hostile.js
//
// 4xx/429 are the point of this test; only 5xx (the API falling over) and
// unreachable hosts count as failures.
import http from 'k6/http'
import { check, sleep } from 'k6'

const BASE = __ENV.BASE_URL || 'http://localhost:8080'

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 499 }))

export const options = {
  scenarios: {
    failed_logins: {
      executor: 'constant-arrival-rate',
      exec: 'failedLogin',
      rate: 60,
      timeUnit: '1m',
      duration: '6m',
      preAllocatedVUs: 10,
      tags: { scenario: 'failed_logins' },
    },
    rate_limit_burst: {
      executor: 'ramping-vus',
      exec: 'burst',
      startVUs: 0,
      startTime: '30s',
      stages: [
        { duration: '15s', target: 50 },
        { duration: '75s', target: 50 },
        { duration: '10s', target: 0 },
      ],
      tags: { scenario: 'rate_limit_burst' },
    },
    refresh_churn: {
      executor: 'constant-arrival-rate',
      exec: 'refreshChurn',
      rate: 12,
      timeUnit: '1m',
      duration: '6m',
      preAllocatedVUs: 5,
      tags: { scenario: 'refresh_churn' },
    },
    invalid_payments: {
      executor: 'constant-arrival-rate',
      exec: 'invalidPayment',
      rate: 12,
      timeUnit: '1m',
      duration: '6m',
      preAllocatedVUs: 5,
      tags: { scenario: 'invalid_payments' },
    },
  },
  thresholds: {
    // The API must survive being attacked: no 5xx, nothing unreachable.
    http_req_failed: ['rate<0.02'],
  },
}

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } }

export function failedLogin() {
  // Mix a real account (wrong password) with unknown emails — both must
  // return the same 401 (or 429 once the auth limiter engages).
  const email =
    Math.random() < 0.5
      ? 'customer@shop.com'
      : `intruder-${Math.floor(Math.random() * 1e6)}@evil.example`
  const res = http.post(
    `${BASE}/api/v1/auth/login`,
    JSON.stringify({ email, password: 'definitely-wrong-password' }),
    JSON_HEADERS,
  )
  check(res, { 'login rejected (401/429)': (r) => r.status === 401 || r.status === 429 })
}

export function burst() {
  // Hammer the general limiter (catalog) and the strict one (login).
  for (let i = 0; i < 5; i++) {
    http.get(`${BASE}/api/v1/products?page_size=5`)
  }
  http.post(
    `${BASE}/api/v1/auth/login`,
    JSON.stringify({ email: 'burst@evil.example', password: 'x' }),
    JSON_HEADERS,
  )
  sleep(0.1)
}

export function refreshChurn() {
  const res = http.post(
    `${BASE}/api/v1/auth/refresh`,
    JSON.stringify({ refresh_token: `garbage-${Math.floor(Math.random() * 1e9)}` }),
    JSON_HEADERS,
  )
  check(res, { 'refresh rejected': (r) => r.status >= 400 })
}

export function invalidPayment() {
  // Unsigned Stripe webhook — exempt from rate limiting (real Stripe retries
  // must never be throttled), so every request exercises signature
  // verification and counts as payments{result="failed",reason="webhook_signature_invalid"}.
  const hook = http.post(
    `${BASE}/api/v1/webhooks/stripe`,
    JSON.stringify({ type: 'payment_intent.succeeded', forged: true }),
    { headers: { 'Content-Type': 'application/json', 'Stripe-Signature': 't=1,v1=forged' } },
  )
  check(hook, { 'forged webhook rejected': (r) => r.status === 400 })

  // Malformed checkout (bad email + missing address) → server-side validation
  // failure, counted as checkout_failures{reason="validation"}.
  const jar = http.cookieJar()
  http.get(`${BASE}/api/v1/cart`) // issues the guest session cookie into the jar
  const res = http.post(
    `${BASE}/api/v1/checkout`,
    JSON.stringify({ email: 'not-an-email', shipping_method: 'standard' }),
    JSON_HEADERS,
  )
  // 422 = server-side validation rejected the payload; 429 = general limiter.
  check(res, { 'bad checkout rejected': (r) => r.status === 422 || r.status === 429 })
  jar.clear(BASE)
}
