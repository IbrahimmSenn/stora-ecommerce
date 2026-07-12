import { useEffect, useState } from 'react'
import { api } from './api'

// Whether the backend runs as a public portfolio demo (DEMO_MODE=true).
// Fetched once per page load and shared across all callers.
let cached: boolean | null = null
let pending: Promise<boolean> | null = null

function fetchDemoMode(): Promise<boolean> {
  if (cached !== null) return Promise.resolve(cached)
  pending ??= api
    .getDemoConfig()
    .then((c) => (cached = c.demo))
    .catch(() => false)
  return pending
}

export function useDemoMode(): boolean {
  const [demo, setDemo] = useState(cached ?? false)
  useEffect(() => {
    let alive = true
    fetchDemoMode().then((d) => {
      if (alive) setDemo(d)
    })
    return () => {
      alive = false
    }
  }, [])
  return demo
}
