import type { ReactNode } from 'react'

type MastheadProps = {
  /** Short eyebrow label above the title, set in small caps. Optional. */
  eyebrow?: string
  /** Page title — set in the display face, weighty. */
  title: ReactNode
  /** Optional caption beneath the title. */
  caption?: ReactNode
  /** Slot for right-aligned content (filters, actions). */
  aside?: ReactNode
  className?: string
}

/**
 * Masthead is the header for a route: a small eyebrow label above a weighty
 * title, with an optional caption and right-aligned aside.
 */
export function Masthead({
  eyebrow,
  title,
  caption,
  aside,
  className = '',
}: MastheadProps) {
  return (
    <header
      className={`mb-8 lg:mb-12 flex items-end justify-between gap-8 ${className}`}
    >
      <div className="min-w-0">
        {eyebrow && (
          <p className="uc-tight text-[0.7rem] text-ink-faint mb-3">{eyebrow}</p>
        )}
        <h1
          className="font-display text-[clamp(1.9rem,5vw,3rem)] leading-[1.0] tracking-[-0.02em] text-ink font-bold"
        >
          {title}
        </h1>
        {caption && (
          <p className="mt-3 text-ink-soft max-w-[60ch] text-[0.95rem] leading-relaxed">
            {caption}
          </p>
        )}
      </div>
      {aside && <div className="shrink-0 self-end">{aside}</div>}
    </header>
  )
}
