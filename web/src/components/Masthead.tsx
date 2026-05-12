import type { ReactNode } from 'react'

type MastheadProps = {
  /** Two-digit editorial marker, eg "01". Optional. */
  number?: string
  /** Short eyebrow label above the title, set in small caps. Optional. */
  eyebrow?: string
  /** Page title — set in the display face, weighty. */
  title: ReactNode
  /** Optional caption beneath the title — calm body face. */
  caption?: ReactNode
  /** Slot for right-aligned content (filters, actions). */
  aside?: ReactNode
  className?: string
}

/**
 * Masthead is the editorial header for a route. Left-aligned, asymmetric;
 * the numeric marker and eyebrow sit above the title in small caps.
 */
export function Masthead({
  number,
  eyebrow,
  title,
  caption,
  aside,
  className = '',
}: MastheadProps) {
  return (
    <header
      className={`mb-12 lg:mb-20 flex items-end justify-between gap-8 ${className}`}
    >
      <div className="min-w-0">
        {(number || eyebrow) && (
          <div className="uc-tight text-[0.7rem] text-ink-faint mb-4 flex items-baseline gap-3">
            {number && <span className="tnum">{number}</span>}
            {number && eyebrow && (
              <span aria-hidden className="text-rule-strong">
                /
              </span>
            )}
            {eyebrow && <span>{eyebrow}</span>}
          </div>
        )}
        <h1
          className="font-display text-[clamp(2.25rem,6vw,4rem)] leading-[0.95] tracking-[-0.02em] text-ink"
          style={{ fontVariationSettings: '"wght" 540, "opsz" 32' }}
        >
          {title}
        </h1>
        {caption && (
          <p className="mt-4 text-ink-soft max-w-[55ch] text-[0.95rem] leading-relaxed">
            {caption}
          </p>
        )}
      </div>
      {aside && <div className="shrink-0 self-end">{aside}</div>}
    </header>
  )
}
