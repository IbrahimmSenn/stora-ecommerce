// Reports Core Web Vitals to the backend, which exposes them as Prometheus
// histograms (Technical Performance dashboard, "Web Vitals (p75)" row).
// sendBeacon so reports survive page unloads and never block navigation.
// Fire-and-forget by design: a lost beacon must never affect the shopper.
import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from 'web-vitals'

function report(metric: Metric) {
  const body = JSON.stringify({ name: metric.name, value: metric.value })
  if (navigator.sendBeacon?.('/api/v1/vitals', body)) return
  // Beacon unavailable (very old browser) or queue full — best-effort fetch.
  fetch('/api/v1/vitals', { method: 'POST', body, keepalive: true }).catch(() => {})
}

export function initVitals() {
  onLCP(report)
  onINP(report)
  onCLS(report)
  onFCP(report)
  onTTFB(report)
}
