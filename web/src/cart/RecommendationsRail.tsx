import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, formatPrice } from '../lib/api'
import type { ProductListItem } from '../lib/api'

export function RecommendationsRail({
  cartVersion,
  limit = 4,
}: {
  cartVersion: number
  limit?: number
}) {
  const [items, setItems] = useState<ProductListItem[] | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .recommendations(limit)
      .then((r) => {
        if (!cancelled) setItems(r.items ?? [])
      })
      .catch(() => {
        if (!cancelled) setItems([])
      })
    return () => {
      cancelled = true
    }
  }, [cartVersion, limit])

  // Hide the rail completely when there's nothing personalised to show —
  // showing an empty section would just be noise.
  if (!items || items.length === 0) return null

  return (
    <section className="mt-20 pt-12 border-t border-rule">
      <header className="mb-8">
        <p className="uc-tight text-[0.7rem] text-ink-faint">
          <span className="tnum">03</span>
          <span aria-hidden className="text-rule-strong mx-2">/</span>
          For you
        </p>
        <h2 className="font-display text-[clamp(1.5rem,3vw,2.25rem)] leading-none mt-3 text-ink">
          Picked from what you've looked at.
        </h2>
      </header>
      <ul className="grid grid-cols-2 md:grid-cols-4 gap-x-8 gap-y-10">
        {items.map((p) => (
          <li key={p.id}>
            <Link
              to={`/product/${p.id}`}
              className="group block focus:outline-none"
            >
              <div className="aspect-square bg-sunken overflow-hidden">
                {p.primary_image ? (
                  <img
                    src={p.primary_image}
                    alt=""
                    loading="lazy"
                    className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
                  />
                ) : (
                  <div className="w-full h-full flex items-center justify-center px-3">
                    <span className="text-[0.7rem] text-ink-faint uc-tight text-center leading-tight line-clamp-3">
                      {p.name}
                    </span>
                  </div>
                )}
              </div>
              <p className="mt-3 text-sm text-ink leading-snug group-hover:text-accent transition-colors">
                {p.name}
              </p>
              <p className="mt-1 text-xs text-ink-faint tnum">
                {formatPrice(p.price)}
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
