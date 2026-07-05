/* RelatedProducts.tsx — "You might also like" rail on the product detail page.
 * Pulls other products from the same category (falling back to the catalogue at
 * large), excluding the product being viewed. Hidden when there's nothing to show.
 */
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, formatPrice, discountPercent } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { StarRating } from '../reviews/StarRating'

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
        {items.map((p) => {
          const off = discountPercent(p.price, p.sale_price)
          const onSale = off != null
          return (
            <article
              key={p.id}
              className="group flex flex-col rounded-lg border border-rule bg-raised p-3 transition-shadow hover:border-rule-strong hover:shadow-[0_6px_20px_oklch(0.2_0.01_265/0.10)]"
            >
              <Link to={`/product/${p.id}`} aria-label={p.name} className="flex flex-col gap-3">
                <div className="relative aspect-square bg-surface rounded-md overflow-hidden p-[6%]">
                  {onSale && (
                    <span className="absolute left-2 top-2 z-10 rounded bg-accent px-1.5 py-0.5 text-[0.7rem] font-semibold text-on-accent tnum">
                      -{off}%
                    </span>
                  )}
                  {p.primary_image ? (
                    <img
                      src={p.primary_image}
                      alt={p.name}
                      loading="lazy"
                      className="w-full h-full object-contain transition-transform duration-300 group-hover:scale-[1.03]"
                    />
                  ) : null}
                </div>
                <div>
                  <h3 className="text-ink text-[0.9rem] leading-snug line-clamp-2 group-hover:text-primary transition-colors">
                    {p.name}
                  </h3>
                  {p.review_count > 0 && (
                    <span className="inline-flex mt-1.5">
                      <StarRating value={p.avg_rating} size={12} count={p.review_count} />
                    </span>
                  )}
                  <div className="mt-1.5">
                    {onSale ? (
                      <span className="tnum text-accent text-base font-bold">{formatPrice(p.sale_price!)}</span>
                    ) : (
                      <span className="tnum text-ink text-base font-bold">{formatPrice(p.price)}</span>
                    )}
                  </div>
                </div>
              </Link>
            </article>
          )
        })}
      </div>
    </section>
  )
}
