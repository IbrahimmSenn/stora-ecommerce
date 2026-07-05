/* MegaSale.tsx — colored "Mega sale" band of discounted products on the home
 * page. Driven by the on-sale list (sorted by biggest discount). A primary
 * band holds a horizontally scrollable strip of detailed white deal cards:
 * photo, discount flag, name, rating + review count, sale price, amount saved,
 * struck price, and stock. Card widths are fixed so the strip scrolls cleanly
 * on mobile and tablet.
 */
import { Link } from 'react-router-dom'
import { formatPrice, discountPercent } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { StarRating } from '../reviews/StarRating'

function stockLine(qty: number) {
  if (qty === 0) return <span className="text-ink-faint">Out of stock</span>
  if (qty <= 5)
    return (
      <span className="text-accent font-medium">
        Only <span className="tnum">{qty}</span> left
      </span>
    )
  return <span className="text-positive font-medium">In stock</span>
}

export function MegaSale({ products }: { products: ProductListItem[] }) {
  if (products.length === 0) return null

  return (
    <section
      aria-labelledby="mega-sale-heading"
      className="mb-10 rounded-xl bg-primary text-on-primary px-4 py-5 sm:px-6 sm:py-6"
    >
      <header className="mb-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2
          id="mega-sale-heading"
          className="font-display text-2xl md:text-3xl font-extrabold tracking-tight text-highlight"
        >
          Mega sale
        </h2>
        <p className="text-sm text-on-primary/80">
          Today's biggest discounts, while stock lasts.
        </p>
      </header>

      <ul className="flex gap-3 sm:gap-4 overflow-x-auto pb-2 [scrollbar-width:thin]">
        {products.map((p) => {
          const off = discountPercent(p.price, p.sale_price)
          const saved = p.sale_price != null ? p.price - p.sale_price : 0
          return (
            <li key={p.id} className="shrink-0 w-44 sm:w-52">
              <Link
                to={`/product/${p.id}`}
                className="group flex h-full flex-col rounded-lg bg-surface text-ink p-3 transition-shadow hover:shadow-[0_8px_24px_oklch(0.2_0.01_265/0.22)]"
              >
                <div className="relative aspect-square rounded-md overflow-hidden bg-surface p-[6%]">
                  {off != null && (
                    <span className="absolute left-1.5 top-1.5 z-10 rounded-full bg-accent px-2 py-1 text-[0.7rem] font-bold leading-none text-on-accent tnum shadow">
                      -{off}%
                    </span>
                  )}
                  {p.primary_image && (
                    <img
                      src={p.primary_image}
                      alt={p.name}
                      loading="lazy"
                      className="h-full w-full object-contain transition-transform duration-300 group-hover:scale-[1.04]"
                    />
                  )}
                </div>

                <h3 className="mt-2.5 text-[0.95rem] font-medium leading-snug text-ink line-clamp-2 group-hover:text-primary transition-colors">
                  {p.name}
                </h3>

                <div className="mt-1 h-4">
                  {p.review_count > 0 && (
                    <StarRating value={p.avg_rating} size={13} count={p.review_count} />
                  )}
                </div>

                <div className="mt-2 flex items-end gap-2 flex-wrap">
                  <span className="tnum text-xl font-extrabold text-ink leading-none">
                    {formatPrice(p.sale_price ?? p.price)}
                  </span>
                  {saved > 0 && (
                    <span className="rounded bg-highlight px-1.5 py-0.5 text-[0.7rem] font-bold text-highlight-ink tnum">
                      Save {formatPrice(saved)}
                    </span>
                  )}
                </div>
                {p.sale_price != null && (
                  <p className="mt-0.5 text-xs text-ink-faint">
                    Normal <span className="tnum line-through">{formatPrice(p.price)}</span>
                  </p>
                )}

                <p className="mt-auto pt-2 text-xs">{stockLine(p.stock_quantity)}</p>
              </Link>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
