/* PromoCarousel.tsx — home hero banner of real discounted products.
 *
 * Each slide features one on-sale product: photo, name, discount badge, sale
 * price, and a CTA to the product. Slides cycle through brand colours. User
 * controls: prev/next arrows, play/pause, and dots. Auto-advance pauses when
 * the user pauses and is off entirely under prefers-reduced-motion.
 * transform/opacity only; never animates layout.
 */
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { formatPrice, discountPercent } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { useReducedMotion } from '../lib/motion'
import { ChevronLeft, ChevronRight, Play, Pause } from '../components/icons'

const THEMES = [
  { surface: 'bg-primary text-on-primary', cta: 'bg-highlight text-highlight-ink hover:brightness-95' },
  { surface: 'bg-accent text-on-accent', cta: 'bg-on-accent text-accent hover:brightness-95' },
  { surface: 'bg-highlight text-highlight-ink', cta: 'bg-primary text-on-primary hover:brightness-110' },
]

const INTERVAL_MS = 5000

export function PromoCarousel({ products }: { products: ProductListItem[] }) {
  const reduced = useReducedMotion()
  const [index, setIndex] = useState(0)
  const [playing, setPlaying] = useState(!reduced)

  const count = products.length

  useEffect(() => {
    if (!playing || reduced || count <= 1) return
    const t = window.setInterval(() => {
      setIndex((i) => (i + 1) % count)
    }, INTERVAL_MS)
    return () => window.clearInterval(t)
  }, [playing, reduced, count])

  if (count === 0) return null

  const go = (next: number) => setIndex(((next % count) + count) % count)

  return (
    <section
      aria-label="Featured deals"
      aria-roledescription="carousel"
      className="relative overflow-hidden rounded-lg h-64 md:h-80"
    >
      {products.map((p, i) => {
        const active = i === index
        const theme = THEMES[i % THEMES.length]
        const off = discountPercent(p.price, p.sale_price)
        return (
          <div
            key={p.id}
            aria-hidden={!active}
            className={`absolute inset-0 grid grid-cols-1 sm:grid-cols-2 items-center gap-4 px-6 md:px-12 ${theme.surface}`}
            style={{
              opacity: active ? 1 : 0,
              transform: reduced ? undefined : `translateX(${active ? '0' : '1.5rem'})`,
              transition: reduced
                ? undefined
                : 'opacity var(--duration-med) var(--ease-out-quart), transform var(--duration-med) var(--ease-out-quart)',
              pointerEvents: active ? 'auto' : 'none',
            }}
          >
            <div className="flex flex-col gap-3 max-w-[34ch]">
              <p className="uc-tight text-[0.7rem] opacity-80">Mega sale</p>
              <h2 className="font-display text-2xl md:text-4xl font-bold leading-tight line-clamp-2">
                {p.name}
              </h2>
              <div className="flex items-baseline gap-3 flex-wrap">
                {off != null && (
                  <span className="rounded-md bg-ink/15 px-2 py-1 text-sm font-bold tnum">
                    Save {off}%
                  </span>
                )}
                <span className="tnum text-xl md:text-2xl font-bold">
                  {formatPrice(p.sale_price ?? p.price)}
                </span>
                {p.sale_price != null && (
                  <span className="tnum line-through opacity-70">{formatPrice(p.price)}</span>
                )}
              </div>
              <Link
                to={`/product/${p.id}`}
                className={`mt-1 inline-flex w-fit items-center rounded-md px-5 py-2.5 text-sm font-medium transition ${theme.cta}`}
              >
                Shop now
              </Link>
            </div>

            <div className="hidden sm:flex items-center justify-center h-full py-6">
              {p.primary_image && (
                <div className="h-full aspect-square max-h-56 rounded-lg bg-surface p-3 shadow-lg">
                  <img
                    src={p.primary_image}
                    alt={p.name}
                    loading={i === 0 ? 'eager' : 'lazy'}
                    className="w-full h-full object-contain"
                  />
                </div>
              )}
            </div>
          </div>
        )
      })}

      {count > 1 && (
        <>
          <button
            type="button"
            onClick={() => go(index - 1)}
            aria-label="Previous deal"
            className="absolute left-2 md:left-3 top-1/2 -translate-y-1/2 inline-flex h-9 w-9 items-center justify-center rounded-full bg-ink/30 text-surface hover:bg-ink/50 transition-colors cursor-pointer"
          >
            <ChevronLeft size={20} aria-hidden />
          </button>
          <button
            type="button"
            onClick={() => go(index + 1)}
            aria-label="Next deal"
            className="absolute right-2 md:right-3 top-1/2 -translate-y-1/2 inline-flex h-9 w-9 items-center justify-center rounded-full bg-ink/30 text-surface hover:bg-ink/50 transition-colors cursor-pointer"
          >
            <ChevronRight size={20} aria-hidden />
          </button>

          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-2 rounded-full bg-ink/30 px-2.5 py-1.5">
            {!reduced && (
              <button
                type="button"
                onClick={() => setPlaying((v) => !v)}
                aria-label={playing ? 'Pause' : 'Play'}
                className="inline-flex h-5 w-5 items-center justify-center text-surface hover:opacity-80 transition-opacity cursor-pointer"
              >
                {playing ? <Pause size={13} aria-hidden /> : <Play size={13} aria-hidden />}
              </button>
            )}
            {products.map((p, i) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setIndex(i)}
                aria-label={`Show deal ${i + 1}`}
                aria-current={i === index}
                className={`h-2 rounded-full transition-all ${
                  i === index ? 'w-6 bg-surface' : 'w-2 bg-surface/50 hover:bg-surface/75'
                }`}
              />
            ))}
          </div>
        </>
      )}
    </section>
  )
}
