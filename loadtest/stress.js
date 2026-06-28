// k6 stress ramp — push browse traffic up to find the knee where latency
// degrades, to answer "max concurrent users before responses exceed 5s".
//   docker run --rm --network host -v "$PWD/loadtest:/s" grafana/k6 run /s/stress.js
import http from 'k6/http'
import { check } from 'k6'

const BASE = __ENV.BASE_URL || 'http://localhost:8080'

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 100 },
        { duration: '15s', target: 300 },
        { duration: '15s', target: 600 },
        { duration: '20s', target: 1000 },
        { duration: '15s', target: 1000 },
        { duration: '5s', target: 0 },
      ],
    },
  },
}

export default function () {
  const res = http.get(`${BASE}/api/v1/products?page_size=20`)
  check(res, { 'ok': (r) => r.status === 200 })
  const id = (() => {
    try {
      const items = res.json('products')
      return items && items.length ? items[0].id : null
    } catch (_) {
      return null
    }
  })()
  if (id) http.get(`${BASE}/api/v1/products/${id}`)
}
