import { useState } from 'react'
import { useDemoMode } from '../lib/useDemoMode'

const DISMISS_KEY = 'demo-banner-dismissed'

// Slim notice shown only on public demo deployments (DEMO_MODE=true): tells
// visitors this is a portfolio store and how to try it.
export function DemoBanner() {
  const demo = useDemoMode()
  const [dismissed, setDismissed] = useState(
    () => localStorage.getItem(DISMISS_KEY) === '1',
  )

  if (!demo || dismissed) return null

  return (
    <div className="bg-sunken border-b border-rule text-xs text-ink-soft">
      <div className="max-w-7xl mx-auto px-4 py-2 flex items-center gap-3">
        <p className="flex-1 leading-relaxed">
          Portfolio demo — payments run in Stripe test mode. Try card{' '}
          <span className="tnum text-ink">4242 4242 4242 4242</span> (any future
          date, any CVC), or sign in as{' '}
          <span className="text-ink">customer@shop.com</span> /{' '}
          <span className="tnum text-ink">customer123</span>.
        </p>
        <button
          type="button"
          onClick={() => {
            localStorage.setItem(DISMISS_KEY, '1')
            setDismissed(true)
          }}
          aria-label="Dismiss demo notice"
          className="shrink-0 px-2 py-1 text-ink-faint hover:text-ink"
        >
          ✕
        </button>
      </div>
    </div>
  )
}
