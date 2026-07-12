import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { ProductCard } from '../products/ProductCard'

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
    <section aria-labelledby="cart-recs-heading" className="mt-20 pt-12 border-t border-rule">
      <header className="mb-8">
        <p className="uc-tight text-[0.7rem] text-ink-faint">For you</p>
        <h2
          id="cart-recs-heading"
          className="font-display text-[clamp(1.5rem,3vw,2.25rem)] leading-none mt-3 text-ink"
        >
          Picked from what you've looked at.
        </h2>
      </header>
      <ul className="grid grid-cols-2 md:grid-cols-4 gap-3 md:gap-5">
        {items.map((p) => (
          <li key={p.id}>
            <ProductCard product={p} />
          </li>
        ))}
      </ul>
    </section>
  )
}
