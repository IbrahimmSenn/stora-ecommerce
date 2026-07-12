/* Countdown.tsx — "deal ends in HH:MM:SS" ticker for the Mega sale header.
 * Cosmetic demo device: it counts to the next local midnight, when the "daily"
 * deals notionally roll over. No backend clock involved.
 */
import { useEffect, useState } from 'react'

function msToMidnight(now: Date): number {
  const midnight = new Date(now)
  midnight.setHours(24, 0, 0, 0)
  return midnight.getTime() - now.getTime()
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function Countdown({ prefix = 'Deals end in' }: { prefix?: string }) {
  const [remaining, setRemaining] = useState(() => msToMidnight(new Date()))

  useEffect(() => {
    const t = window.setInterval(() => setRemaining(msToMidnight(new Date())), 1000)
    return () => window.clearInterval(t)
  }, [])

  const totalSec = Math.max(0, Math.floor(remaining / 1000))
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60

  // aria-label only carries hours/minutes so screen readers aren't offered a
  // per-second churn; the visible seconds are aria-hidden alongside it.
  return (
    <span role="timer" aria-live="off" aria-label={`${prefix} ${h} hours ${m} minutes`}>
      {prefix}{' '}
      <span aria-hidden className="tnum font-bold">
        {pad(h)}:{pad(m)}:{pad(s)}
      </span>
    </span>
  )
}
