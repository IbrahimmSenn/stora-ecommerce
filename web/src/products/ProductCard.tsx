/* ProductCard.tsx — the one product card used everywhere: catalogue grid,
 * horizontal rails (Mega sale, best sellers, recommendations), and list view.
 * Variants only change the frame; badges, pricing, rating, and stock read the
 * same in every context.
 */
import { Link } from 'react-router-dom'
import { formatPrice, discountPercent } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { StarRating } from '../reviews/StarRating'
import { Plus } from '../components/icons'

export type ProductCardVariant = 'grid' | 'rail' | 'list'

export function StockSignal({ qty }: { qty: number }) {
  if (qty === 0) return <span className="uc-tight text-[0.7rem] text-ink-faint italic">Out of stock</span>
  if (qty <= 5)
    return (
      <span className="uc-tight text-[0.7rem] text-accent">
        Only <span className="tnum">{qty}</span> left
      </span>
    )
  return <span className="uc-tight text-[0.7rem] text-positive">In stock</span>
}

function QuickAddButton({
  productName,
  busy,
  disabled,
  onClick,
}: {
  productName: string
  busy: boolean
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || busy}
      aria-label={disabled ? `${productName} is unavailable` : `Add ${productName} to cart`}
      title={disabled ? 'Unavailable' : 'Add to cart'}
      className="inline-flex h-9 w-9 items-center justify-center border border-rule text-ink hover:border-accent hover:text-accent transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:border-rule disabled:hover:text-ink"
    >
      <Plus
        size={16}
        strokeWidth={1.5}
        aria-hidden
        style={{ opacity: busy ? 0.4 : 1, transition: 'opacity 120ms var(--ease-out-quart)' }}
      />
    </button>
  )
}

function SaleFlag({ off }: { off: number }) {
  return (
    <span className="absolute left-1.5 top-1.5 z-10 rounded-full bg-accent px-2 py-1 text-[0.7rem] font-bold leading-none text-on-accent tnum shadow">
      -{off}%
    </span>
  )
}

function CardBadge({ badge }: { badge: 'bestseller' | 'new' }) {
  return (
    <span className="absolute right-1.5 top-1.5 z-10 rounded bg-highlight px-1.5 py-0.5 text-[0.65rem] font-bold uppercase tracking-wide text-highlight-ink shadow">
      {badge === 'bestseller' ? 'Bestseller' : 'New'}
    </span>
  )
}

function PriceBlock({ product, size = 'lg' }: { product: ProductListItem; size?: 'md' | 'lg' }) {
  const onSale = product.sale_price != null
  const priceCls = size === 'lg' ? 'text-lg' : 'text-base'
  return (
    <div className="min-w-0">
      <div className="flex items-end gap-1.5 flex-wrap">
        <span className={`tnum ${priceCls} font-extrabold leading-none ${onSale ? 'text-accent' : 'text-ink'}`}>
          {formatPrice(product.sale_price ?? product.price)}
        </span>
        {onSale && (
          <span className="rounded bg-highlight px-1.5 py-0.5 text-[0.7rem] font-bold text-highlight-ink tnum">
            Save {formatPrice(product.price - product.sale_price!)}
          </span>
        )}
      </div>
      {onSale && (
        <p className="mt-0.5 text-xs text-ink-faint">
          Normal <span className="tnum line-through">{formatPrice(product.price)}</span>
        </p>
      )}
    </div>
  )
}

export function ProductCard({
  product,
  variant = 'grid',
  busy = false,
  onAdd,
  badge,
}: {
  product: ProductListItem
  variant?: ProductCardVariant
  busy?: boolean
  /** When provided, the card shows a quick-add button. */
  onAdd?: () => void
  badge?: 'bestseller' | 'new'
}) {
  const off = discountPercent(product.price, product.sale_price)
  const onSale = off != null

  const image = (
    <div className="relative aspect-square bg-surface rounded-md overflow-hidden p-[6%]">
      {onSale && <SaleFlag off={off} />}
      {badge && <CardBadge badge={badge} />}
      {product.primary_image ? (
        <img
          src={product.primary_image}
          alt={product.name}
          loading="lazy"
          className="w-full h-full object-contain transition-transform duration-300 group-hover:scale-[1.04]"
        />
      ) : null}
    </div>
  )

  if (variant === 'list') {
    return (
      <article className="group flex gap-4 sm:gap-6 rounded-lg border border-rule bg-raised p-3 transition-shadow hover:border-rule-strong hover:shadow-[0_6px_20px_oklch(0.2_0.01_265/0.10)]">
        <Link
          to={`/product/${product.id}`}
          aria-label={product.name}
          className="relative h-24 w-24 sm:h-28 sm:w-28 shrink-0 bg-surface rounded-md overflow-hidden p-[6%]"
        >
          {onSale && <SaleFlag off={off} />}
          {product.primary_image ? (
            <img
              src={product.primary_image}
              alt={product.name}
              loading="lazy"
              className="w-full h-full object-contain transition-transform duration-300 group-hover:scale-[1.04]"
            />
          ) : null}
        </Link>

        <div className="flex-1 min-w-0 flex flex-col">
          {product.brand_name && (
            <p className="text-xs text-ink-faint uppercase tracking-wide">{product.brand_name}</p>
          )}
          <Link to={`/product/${product.id}`} className="min-w-0">
            <h3 className="text-ink text-[1rem] leading-snug line-clamp-2 group-hover:text-primary transition-colors">
              {product.name}
            </h3>
          </Link>
          {product.review_count > 0 && (
            <span className="inline-flex mt-1.5">
              <StarRating value={product.avg_rating} size={13} count={product.review_count} />
            </span>
          )}
          <div className="mt-1.5">
            <StockSignal qty={product.stock_quantity} />
          </div>
        </div>

        <div className="shrink-0 flex flex-col items-end justify-between gap-2">
          <div className="text-right">
            <PriceBlock product={product} />
          </div>
          {onAdd && (
            <QuickAddButton
              productName={product.name}
              busy={busy}
              disabled={product.stock_quantity === 0}
              onClick={onAdd}
            />
          )}
        </div>
      </article>
    )
  }

  // grid and rail share the vertical layout; rail pins the card width so
  // horizontal strips scroll cleanly.
  const frame =
    variant === 'rail'
      ? 'w-44 sm:w-52 bg-raised border border-rule'
      : 'bg-raised border border-rule hover:border-rule-strong'

  return (
    <article
      className={`group flex h-full flex-col rounded-lg p-3 transition-shadow hover:shadow-[0_8px_24px_oklch(0.2_0.01_265/0.18)] ${frame}`}
    >
      <Link to={`/product/${product.id}`} aria-label={product.name} className="flex flex-col gap-2.5">
        {image}
        <div>
          {product.brand_name && (
            <p className="text-xs text-ink-faint uppercase tracking-wide">{product.brand_name}</p>
          )}
          <h3 className="text-ink text-[0.95rem] leading-snug line-clamp-2 group-hover:text-primary transition-colors">
            {product.name}
          </h3>
          {/* Fixed-height slot so cards align whether or not reviews exist. */}
          <div className="mt-1 h-4">
            {product.review_count > 0 && (
              <StarRating value={product.avg_rating} size={13} count={product.review_count} />
            )}
          </div>
        </div>
      </Link>

      <div className="mt-auto pt-2 flex items-end justify-between gap-2">
        <div className="min-w-0">
          <PriceBlock product={product} />
          <div className="mt-1.5">
            <StockSignal qty={product.stock_quantity} />
          </div>
        </div>
        {onAdd && (
          <QuickAddButton
            productName={product.name}
            busy={busy}
            disabled={product.stock_quantity === 0}
            onClick={onAdd}
          />
        )}
      </div>
    </article>
  )
}
