/* Rail.tsx — horizontal product strip with scroll snapping, edge fades, and
 * prev/next buttons. Tones: plain (transparent), soft (sunken band), loud
 * (primary band, light-on-dark header). Children are the <li> cards.
 */
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ChevronLeft, ChevronRight } from './icons'
import { useReducedMotion } from '../lib/motion'

export type RailTone = 'plain' | 'soft' | 'loud'

const TONE_SECTION: Record<RailTone, string> = {
  plain: '',
  soft: 'rounded-xl bg-sunken px-4 py-5 sm:px-6 sm:py-6',
  loud: 'rounded-xl bg-primary text-on-primary px-4 py-5 sm:px-6 sm:py-6',
}

export function Rail({
  id,
  title,
  eyebrow,
  action,
  tone = 'plain',
  children,
}: {
  /** Used for the heading id / aria-labelledby pair. */
  id: string
  title: ReactNode
  /** Small line above or beside the title (e.g. countdown, tagline). */
  eyebrow?: ReactNode
  /** Right-aligned header slot, e.g. a "See all" link. */
  action?: ReactNode
  tone?: RailTone
  children: ReactNode
}) {
  const scrollerRef = useRef<HTMLUListElement>(null)
  const [canBack, setCanBack] = useState(false)
  const [canForward, setCanForward] = useState(false)
  const reduced = useReducedMotion()

  useEffect(() => {
    const el = scrollerRef.current
    if (!el) return
    const update = () => {
      setCanBack(el.scrollLeft > 4)
      setCanForward(el.scrollLeft + el.clientWidth < el.scrollWidth - 4)
    }
    update()
    el.addEventListener('scroll', update, { passive: true })
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => {
      el.removeEventListener('scroll', update)
      ro.disconnect()
    }
  }, [children])

  function scrollByPage(dir: 1 | -1) {
    const el = scrollerRef.current
    if (!el) return
    el.scrollBy({ left: dir * el.clientWidth * 0.8, behavior: reduced ? 'auto' : 'smooth' })
  }

  const headingCls =
    tone === 'loud'
      ? 'font-display text-2xl md:text-3xl font-extrabold tracking-tight text-highlight'
      : 'font-display text-2xl md:text-3xl font-extrabold tracking-tight text-ink'

  const arrowCls =
    tone === 'loud'
      ? 'border-on-primary/30 text-on-primary hover:border-on-primary disabled:hover:border-on-primary/30'
      : 'border-rule text-ink-soft hover:border-ink hover:text-ink disabled:hover:border-rule disabled:hover:text-ink-soft'

  return (
    <section aria-labelledby={`${id}-heading`} className={TONE_SECTION[tone]}>
      <header className="mb-4 flex flex-wrap items-end justify-between gap-x-4 gap-y-1">
        <div className="min-w-0">
          <h2 id={`${id}-heading`} className={headingCls}>
            {title}
          </h2>
          {eyebrow && (
            <div className={`mt-1 text-sm ${tone === 'loud' ? 'text-on-primary/80' : 'text-ink-soft'}`}>
              {eyebrow}
            </div>
          )}
        </div>
        <div className="flex items-center gap-3">
          {action}
          <div className="hidden sm:flex items-center gap-1">
            <button
              type="button"
              onClick={() => scrollByPage(-1)}
              disabled={!canBack}
              aria-label="Scroll back"
              className={`inline-flex h-8 w-8 items-center justify-center rounded-full border transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed ${arrowCls}`}
            >
              <ChevronLeft size={16} strokeWidth={1.5} aria-hidden />
            </button>
            <button
              type="button"
              onClick={() => scrollByPage(1)}
              disabled={!canForward}
              aria-label="Scroll forward"
              className={`inline-flex h-8 w-8 items-center justify-center rounded-full border transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed ${arrowCls}`}
            >
              <ChevronRight size={16} strokeWidth={1.5} aria-hidden />
            </button>
          </div>
        </div>
      </header>

      <div className="relative">
        {/* Edge fades hint at more content without blocking clicks. */}
        {canBack && (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0 left-0 z-10 w-8"
            style={{ background: `linear-gradient(to right, ${tone === 'loud' ? 'var(--color-primary)' : tone === 'soft' ? 'var(--color-sunken)' : 'var(--color-surface)'}, transparent)` }}
          />
        )}
        {canForward && (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0 right-0 z-10 w-8"
            style={{ background: `linear-gradient(to left, ${tone === 'loud' ? 'var(--color-primary)' : tone === 'soft' ? 'var(--color-sunken)' : 'var(--color-surface)'}, transparent)` }}
          />
        )}
        <ul
          ref={scrollerRef}
          className="flex gap-3 sm:gap-4 overflow-x-auto pb-2 snap-x snap-proximity [scrollbar-width:thin]"
        >
          {children}
        </ul>
      </div>
    </section>
  )
}
