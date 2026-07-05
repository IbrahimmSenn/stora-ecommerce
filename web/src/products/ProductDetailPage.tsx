/* ProductDetailPage.tsx — /product/:id
 *
 * Editorial three-column layout: image stack on the left, name + price +
 * description + specs in the centre, and a sticky purchase column on the
 * right with quantity stepper and Add to Cart. The Add path triggers the
 * existing CartPanel slide-in (the signature cart transition belongs to a
 * deliberate purchase action, not the quick-add on cards).
 */
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, ApiError, formatPrice, discountPercent } from '../lib/api'
import type { ProductDetail, ProductImage } from '../lib/api'
import { useCart } from '../cart/useCart'
import { useCartPanel } from '../cart/useCartPanel'
import { Page } from '../components/Page'
import { Seo } from '../components/Seo'
import { Skeleton } from '../components/Skeleton'
import { Minus, Plus } from '../components/icons'
import { StarRating } from '../reviews/StarRating'
import { ReviewsSection } from '../reviews/ReviewsSection'
import { RelatedProducts } from './RelatedProducts'

function StockSignal({ qty }: { qty: number }) {
  if (qty === 0)
    return (
      <span className="uc-tight text-[0.7rem] text-ink-faint italic">
        Out of stock.
      </span>
    )
  if (qty <= 5)
    return (
      <span className="uc-tight text-[0.7rem] text-accent">
        Only <span className="tnum">{qty}</span> left.
      </span>
    )
  return (
    <span className="uc-tight text-[0.7rem] text-ink-faint">
      <span className="tnum">{qty}</span> in stock.
    </span>
  )
}

function sortImages(images: ProductImage[]): ProductImage[] {
  return [...images].sort((a, b) => {
    if (a.is_primary === b.is_primary) return 0
    return a.is_primary ? -1 : 1
  })
}

export function ProductDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { addItem } = useCart()
  const { openWith } = useCartPanel()

  const [product, setProduct] = useState<ProductDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [imageIdx, setImageIdx] = useState(0)
  const [quantity, setQuantity] = useState(1)
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    let cancelled = false
    // Reset loading + error on id change so the new fetch starts from a
    // clean slate. The lint rule warns about setState-in-effect; here it's
    // the right shape — we're reacting to an external input (the URL :id).
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)
    setLoadError(null)
    api
      .getProduct(id)
      .then((p) => {
        if (cancelled) return
        setProduct(p)
        setImageIdx(0)
        setQuantity(1)
      })
      .catch((e) => {
        if (cancelled) return
        if (e instanceof ApiError && e.status === 404) {
          setLoadError('Product not found.')
        } else {
          setLoadError(e instanceof Error ? e.message : 'Could not load product.')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  if (loading) {
    return (
      <Page>
        <div
          aria-busy="true"
          aria-label="Loading product"
          className="grid grid-cols-1 md:grid-cols-12 gap-x-10 lg:gap-x-16 gap-y-12"
        >
          <div className="md:col-span-6 lg:col-span-5">
            <Skeleton className="aspect-[4/5] w-full rounded-none" />
          </div>
          <div className="md:col-span-6 lg:col-span-7 space-y-4">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-8 w-3/4" />
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-px w-full" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-11 w-40" />
          </div>
        </div>
      </Page>
    )
  }

  if (loadError || !product) {
    return (
      <Page>
        <Seo title="Product not found" noindex />
        <div className="flex flex-col gap-6">
          <p className="text-sm text-accent">{loadError ?? 'Product not found.'}</p>
          <Link to="/" className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors">
            Back to shop
          </Link>
        </div>
      </Page>
    )
  }

  const images = sortImages(product.images)
  const mainImage = images[imageIdx]
  const maxQty = Math.max(1, Math.min(product.stock_quantity, 99))
  const inStock = product.stock_quantity > 0

  async function handleAdd(trigger: HTMLButtonElement) {
    if (!product || !inStock) return
    setAdding(true)
    setAddError(null)
    try {
      await addItem(product.id, quantity)
      openWith(
        {
          productId: product.id,
          productName: product.name,
          unitPriceCents: product.sale_price ?? product.price,
          quantity,
          imageUrl: images[0]?.url ?? null,
        },
        trigger,
      )
    } catch (e) {
      setAddError(e instanceof Error ? e.message : 'Could not add to cart.')
    } finally {
      setAdding(false)
    }
  }

  const metaDesc = (product.description ?? `${product.name} — available now at Stora.`)
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 157)

  return (
    <Page>
      <Seo title={product.name.slice(0, 50)} description={metaDesc} />
      <nav aria-label="Breadcrumb" className="uc-tight text-[0.7rem] text-ink-faint mb-12 flex flex-wrap items-center gap-x-2">
        <Link to="/" className="hover:text-ink transition-colors">
          Shop
        </Link>
        {product.category_name && (
          <>
            <span aria-hidden className="text-rule-strong">
              /
            </span>
            {product.category_slug ? (
              <Link
                to={`/shop/${product.category_slug}`}
                className="hover:text-ink transition-colors"
              >
                {product.category_name}
              </Link>
            ) : (
              <span>{product.category_name}</span>
            )}
          </>
        )}
        <span aria-hidden className="text-rule-strong">
          /
        </span>
        <span className="text-ink-soft normal-case tracking-normal truncate max-w-[24ch]">
          {product.name}
        </span>
      </nav>

      <div className="grid grid-cols-1 md:grid-cols-12 gap-x-10 lg:gap-x-16 gap-y-12">
        {/* Image stack */}
        <section className="md:col-span-6 lg:col-span-5 flex flex-col gap-4">
          <div className="aspect-[4/5] bg-sunken overflow-hidden">
            {mainImage ? (
              <img
                key={mainImage.id}
                src={mainImage.full_url ?? mainImage.url}
                alt={`${product.name} — view ${imageIdx + 1}`}
                className="w-full h-full object-cover"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center px-8">
                <p className="text-sm text-ink-faint text-center">{product.name}</p>
              </div>
            )}
          </div>
          {images.length > 1 && (
            <ul className="flex flex-wrap gap-2">
              {images.map((img, i) => (
                <li key={img.id}>
                  <button
                    type="button"
                    onClick={() => setImageIdx(i)}
                    aria-label={`Show image ${i + 1} of ${images.length}`}
                    aria-pressed={i === imageIdx}
                    className={`block aspect-square w-16 bg-sunken overflow-hidden cursor-pointer border transition-colors ${
                      i === imageIdx
                        ? 'border-accent'
                        : 'border-transparent hover:border-rule-strong'
                    }`}
                  >
                    <img src={img.thumbnail_url ?? img.url} alt="" className="w-full h-full object-cover" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* Information */}
        <section className="md:col-span-6 lg:col-span-4 flex flex-col gap-6">
          <div>
            <h1 className="font-display text-[clamp(1.75rem,3.5vw,2.5rem)] leading-tight text-ink font-bold">
              {product.name}
            </h1>
            {product.brand_name && (
              <p className="text-sm text-ink-soft mt-2">{product.brand_name}</p>
            )}
            {product.review_count > 0 ? (
              <a href="#reviews-heading" className="inline-flex mt-3 group/rating">
                <StarRating
                  value={product.avg_rating}
                  size={15}
                  count={product.review_count}
                  className="group-hover/rating:[&_*]:text-accent"
                />
              </a>
            ) : (
              <p className="uc-tight text-[0.7rem] text-ink-faint mt-3">No reviews yet.</p>
            )}
          </div>

          {product.description && (
            <p className="text-ink-soft leading-relaxed whitespace-pre-line">
              {product.description}
            </p>
          )}

          {(product.brand_name ||
            product.weight_g != null ||
            product.dimensions_cm != null) && (
            <dl className="border-t border-rule pt-6 flex flex-col gap-3 text-sm">
              {product.brand_name && (
                <SpecRow label="Brand" value={product.brand_name} />
              )}
              {product.weight_g != null && (
                <SpecRow
                  label="Weight"
                  value={
                    <>
                      <span className="tnum">{product.weight_g}</span> g
                    </>
                  }
                />
              )}
              {product.dimensions_cm != null && (
                <SpecRow
                  label="Dimensions"
                  value={
                    <>
                      <span className="tnum">{product.dimensions_cm}</span> cm
                    </>
                  }
                />
              )}
            </dl>
          )}
        </section>

        {/* Sticky purchase column */}
        <aside className="md:col-span-12 lg:col-span-3 md:sticky md:top-24 md:self-start">
          <div className="flex flex-col gap-6 border-t border-rule pt-6 lg:border-t-0 lg:pt-0">
            {(() => {
              const off = discountPercent(product.price, product.sale_price)
              if (off == null) {
                return (
                  <p className="font-display tnum text-ink text-[clamp(1.5rem,2.5vw,2rem)] leading-none font-bold">
                    {formatPrice(product.price)}
                  </p>
                )
              }
              return (
                <div className="flex items-baseline gap-3 flex-wrap">
                  <p className="font-display tnum text-accent text-[clamp(1.5rem,2.5vw,2rem)] leading-none font-bold">
                    {formatPrice(product.sale_price!)}
                  </p>
                  <p className="tnum text-ink-faint line-through">
                    {formatPrice(product.price)}
                  </p>
                  <span className="rounded bg-accent px-1.5 py-0.5 text-[0.7rem] font-semibold text-on-accent tnum">
                    -{off}%
                  </span>
                </div>
              )
            })()}

            <div className="flex flex-col gap-2">
              <label className="uc-tight text-[0.7rem] text-ink-faint">
                Quantity
              </label>
              <div className="inline-flex items-center border border-rule self-start">
                <button
                  type="button"
                  onClick={() => setQuantity((q) => Math.max(1, q - 1))}
                  disabled={quantity <= 1 || !inStock}
                  aria-label="Decrease quantity"
                  className="h-11 w-11 inline-flex items-center justify-center text-ink hover:text-accent transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <Minus size={14} strokeWidth={1.5} aria-hidden />
                </button>
                <span
                  aria-live="polite"
                  className="tnum text-ink min-w-[2.5ch] text-center select-none"
                >
                  {quantity}
                </span>
                <button
                  type="button"
                  onClick={() => setQuantity((q) => Math.min(maxQty, q + 1))}
                  disabled={quantity >= maxQty || !inStock}
                  aria-label="Increase quantity"
                  className="h-11 w-11 inline-flex items-center justify-center text-ink hover:text-accent transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <Plus size={14} strokeWidth={1.5} aria-hidden />
                </button>
              </div>
            </div>

            <button
              type="button"
              onClick={(e) => handleAdd(e.currentTarget)}
              disabled={!inStock || adding}
              className="bg-accent hover:bg-accent-soft text-on-accent transition-colors px-5 py-3 text-sm tracking-[0.01em] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            >
              {adding ? 'Adding.' : inStock ? 'Add to cart' : 'Out of stock'}
            </button>

            <StockSignal qty={product.stock_quantity} />

            {addError && (
              <p className="text-xs text-accent" role="alert">
                {addError}
              </p>
            )}

            <button
              type="button"
              onClick={() => navigate(-1)}
              className="text-xs text-ink-faint hover:text-ink underline underline-offset-4 decoration-rule-strong self-start cursor-pointer"
            >
              Back
            </button>
          </div>
        </aside>
      </div>

      <RelatedProducts productId={product.id} categoryId={product.category_id} />

      <ReviewsSection productId={product.id} />
    </Page>
  )
}

function SpecRow({
  label,
  value,
}: {
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="flex items-baseline justify-between gap-6">
      <dt className="uc-tight text-[0.7rem] text-ink-faint">{label}</dt>
      <dd className="text-ink">{value}</dd>
    </div>
  )
}
