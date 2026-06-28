// k6 load test — three real-user scenarios against the API.
//   1. browse        — catalogue browsing (PLP → PDP → reviews → categories)
//   2. search_cart    — search, then add an item to the (guest) cart
//   3. checkout       — guest checkout placing a real order
//
// Run (stack already up, rate limits relaxed via the loadtest override):
//   docker run --rm --network host -v "$PWD/loadtest:/s" grafana/k6 run /s/load.js
import http from 'k6/http'
import { check, sleep } from 'k6'

const BASE = __ENV.BASE_URL || 'http://localhost:8080'
// High-stock seed product so checkout isn't starved by inventory depletion.
const CHECKOUT_PRODUCT = __ENV.CHECKOUT_PRODUCT || 'a0000000-0000-0000-0000-00000000002a'
const SEARCH_TERMS = ['headphones', 'chair', 'table', 'sleek', 'modern', 'wireless']

export const options = {
  scenarios: {
    browse: {
      executor: 'ramping-vus',
      exec: 'browse',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 50 }, // ramp to 50 concurrent
        { duration: '40s', target: 50 }, // hold (target: 50 users, no degradation)
        { duration: '10s', target: 0 },
      ],
      tags: { scenario: 'browse' },
    },
    search_cart: {
      executor: 'ramping-vus',
      exec: 'searchCart',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 25 },
        { duration: '40s', target: 25 },
        { duration: '10s', target: 0 },
      ],
      tags: { scenario: 'search_cart' },
    },
    checkout: {
      executor: 'constant-vus',
      exec: 'checkout',
      vus: 8,
      duration: '70s',
      tags: { scenario: 'checkout' },
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'], // <5% errors overall
    http_req_duration: ['p(90)<2000', 'p(95)<2000'],
    'http_req_duration{scenario:browse}': ['p(95)<2000'],
    'http_req_duration{scenario:search_cart}': ['p(95)<2000'],
    'http_req_duration{scenario:checkout}': ['p(95)<3000'],
  },
}

export function browse() {
  const list = http.get(`${BASE}/api/v1/products?page_size=20`)
  check(list, { 'products 200': (r) => r.status === 200 })
  let id = CHECKOUT_PRODUCT
  try {
    const items = list.json('products')
    if (items && items.length) id = items[Math.floor(Math.random() * items.length)].id
  } catch (_) { /* keep default */ }

  http.get(`${BASE}/api/v1/products/${id}`)
  http.get(`${BASE}/api/v1/products/${id}/reviews`)
  http.get(`${BASE}/api/v1/categories`)
  sleep(Math.random() * 2 + 0.5)
}

export function searchCart() {
  const q = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)]
  const res = http.get(`${BASE}/api/v1/products?q=${q}&page_size=10`)
  check(res, { 'search 200': (r) => r.status === 200 })

  http.get(`${BASE}/api/v1/cart`) // issues the guest_session cookie
  const add = http.post(
    `${BASE}/api/v1/cart/items`,
    JSON.stringify({ product_id: CHECKOUT_PRODUCT, quantity: 1 }),
    { headers: { 'Content-Type': 'application/json' } },
  )
  check(add, { 'add to cart ok': (r) => r.status === 200 })
  sleep(Math.random() * 2 + 0.5)
}

export function checkout() {
  http.get(`${BASE}/api/v1/cart`) // guest cookie
  http.post(
    `${BASE}/api/v1/cart/items`,
    JSON.stringify({ product_id: CHECKOUT_PRODUCT, quantity: 1 }),
    { headers: { 'Content-Type': 'application/json' } },
  )
  const order = http.post(
    `${BASE}/api/v1/checkout`,
    JSON.stringify({
      email: 'loadtest@example.com',
      phone: '+3725551234',
      shipping_method: 'standard',
      address_override: true,
      address: {
        recipient_name: 'Load Tester',
        line1: '1 Test Street',
        city: 'Tallinn',
        region: 'Harju',
        postal_code: '10115',
        country: 'EE',
      },
    }),
    { headers: { 'Content-Type': 'application/json' } },
  )
  check(order, { 'order placed': (r) => r.status === 201 })
  sleep(Math.random() * 1.5 + 0.5)
}
