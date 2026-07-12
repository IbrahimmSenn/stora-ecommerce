/* AnnouncementBar.tsx — slim marketing strip above the nav. Dismissal lasts
 * for the browser session. Distinct from DemoBanner (environment notice).
 */
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { X } from './icons'

const DISMISS_KEY = 'stora-announcement-dismissed'

export function AnnouncementBar() {
  const [dismissed, setDismissed] = useState(() => {
    try {
      return sessionStorage.getItem(DISMISS_KEY) === '1'
    } catch {
      return false
    }
  })

  if (dismissed) return null

  function dismiss() {
    setDismissed(true)
    try {
      sessionStorage.setItem(DISMISS_KEY, '1')
    } catch {
      // Session storage unavailable (private mode) — dismissal just won't stick.
    }
  }

  return (
    <div className="bg-highlight text-highlight-ink">
      <div className="mx-auto flex max-w-7xl items-center justify-center gap-2 px-10 py-1.5 text-center text-sm font-medium relative">
        <p>
          Mega sale is on — up to 60% off.{' '}
          <Link to="/?sort=discount" className="underline underline-offset-2 hover:opacity-80 transition-opacity">
            See all deals
          </Link>{' '}
          · Free shipping over $50
        </p>
        <button
          type="button"
          onClick={dismiss}
          aria-label="Dismiss announcement"
          className="absolute right-2 top-1/2 -translate-y-1/2 inline-flex h-6 w-6 items-center justify-center rounded hover:bg-highlight-ink/10 transition-colors cursor-pointer"
        >
          <X size={14} strokeWidth={2} aria-hidden />
        </button>
      </div>
    </div>
  )
}
