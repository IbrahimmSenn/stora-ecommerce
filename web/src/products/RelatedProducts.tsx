/* RelatedProducts.tsx — "You might also like" rail on the product detail page.
 * Pulls other products from the same category (falling back to the catalogue at
 * large), excluding the product being viewed. Hidden when there's nothing to show.
 */
import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { ProductCard } from './ProductCard'

export function RelatedProducts({
  productId,
  categoryId,
}: {
  productId: string
  categoryId?: string | null
}) {
  const [items, setItems] = useState<ProductListItem[] | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .listProducts({ categoryId: categoryId ?? undefined, pageSize: 12, sort: 'rating' })
      .then((res) => {
        if (cancelled) return
        setItems(res.products.filter((p) => p.id !== productId).slice(0, 5))
      })
      .catch(() => {
        if (!cancelled) setItems([])
      })
    return () => {
      cancelled = true
    }
  }, [productId, categoryId])

  if (!items || items.length === 0) return null

  return (
    <section aria-labelledby="related-heading" className="border-t border-rule pt-12 mt-16">
      <header className="flex flex-col gap-1 mb-8">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Related</span>
        <h2 id="related-heading" className="font-display text-[clamp(1.5rem,3vw,2rem)] leading-tight text-ink font-bold">
          You might also like.
        </h2>
      </header>

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 md:gap-5">
        {items.map((p) => (
          <ProductCard key={p.id} product={p} />
        ))}
      </div>
    </section>
  )
}
