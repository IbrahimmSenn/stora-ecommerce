import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api, ApiError, formatPrice, discountPercent } from '../lib/api'
import type { Category, ProductListItem } from '../lib/api'
import { useCart } from '../cart/useCart'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Plus } from '../components/icons'
import { useToast } from '../components/useToast'
import { StarRating } from '../reviews/StarRating'
import { Seo } from '../components/Seo'
import { PromoCarousel } from './PromoCarousel'
import { MegaSale } from './MegaSale'

function StockSignal({ qty }: { qty: number }) {
  if (qty === 0) return <span className="uc-tight text-[0.7rem] text-ink-faint italic">Out of stock</span>
  if (qty <= 5)
    return (
      <span className="uc-tight text-[0.7rem] text-accent">
        Only <span className="tnum">{qty}</span> left
      </span>
    )
  return <span className="uc-tight text-[0.7rem] text-ink-faint"><span className="tnum">{qty}</span> in stock</span>
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

function ProductCard({
  product,
  busy,
  onAdd,
}: {
  product: ProductListItem
  busy: boolean
  onAdd: () => void
}) {
  const off = discountPercent(product.price, product.sale_price)
  const onSale = off != null
  return (
    <article className="group flex flex-col rounded-lg border border-rule bg-raised p-3 transition-shadow hover:border-rule-strong hover:shadow-[0_6px_20px_oklch(0.2_0.01_265/0.10)]">
      <Link
        to={`/product/${product.id}`}
        aria-label={product.name}
        className="flex flex-col gap-3"
      >
        <div className="relative aspect-square bg-surface rounded-md overflow-hidden p-[6%]">
          {onSale && (
            <span className="absolute left-2 top-2 z-10 rounded bg-accent px-1.5 py-0.5 text-[0.7rem] font-semibold text-on-accent tnum">
              -{off}%
            </span>
          )}
          {product.primary_image ? (
            <img
              src={product.primary_image}
              alt={product.name}
              loading="lazy"
              className="w-full h-full object-contain transition-transform duration-300 group-hover:scale-[1.03]"
            />
          ) : null}
        </div>
        <div>
          {product.brand_name && (
            <p className="text-xs text-ink-faint uppercase tracking-wide">{product.brand_name}</p>
          )}
          <h3 className="text-ink text-[0.95rem] leading-snug line-clamp-2 group-hover:text-primary transition-colors">
            {product.name}
          </h3>
          {product.review_count > 0 && (
            <span className="inline-flex mt-1.5">
              <StarRating value={product.avg_rating} size={13} count={product.review_count} />
            </span>
          )}
        </div>
      </Link>

      <div className="mt-auto pt-3 flex items-end justify-between gap-2">
        <div className="min-w-0">
          {onSale ? (
            <>
              <div className="flex items-baseline gap-1.5 flex-wrap">
                <span className="tnum text-accent text-lg font-bold">
                  {formatPrice(product.sale_price!)}
                </span>
                <span className="tnum text-ink-faint line-through text-xs">
                  {formatPrice(product.price)}
                </span>
              </div>
              <span className="mt-1 inline-block rounded bg-highlight px-1.5 py-0.5 text-[0.7rem] font-bold text-highlight-ink tnum">
                Save {formatPrice(product.price - product.sale_price!)}
              </span>
            </>
          ) : (
            <p className="tnum text-ink text-lg font-bold">{formatPrice(product.price)}</p>
          )}
          <div className="mt-1.5">
            <StockSignal qty={product.stock_quantity} />
          </div>
        </div>
        <QuickAddButton
          productName={product.name}
          busy={busy}
          disabled={product.stock_quantity === 0}
          onClick={onAdd}
        />
      </div>
    </article>
  )
}

type Mode =
  | { kind: 'all' }
  | { kind: 'category'; category: Category }
  | { kind: 'search'; query: string }

export function ProductsPage() {
  const { slug } = useParams<{ slug?: string }>()
  const [searchParams] = useSearchParams()
  const rawQuery = searchParams.get('q')?.trim() ?? ''

  const { addItem } = useCart()
  const { show: showToast } = useToast()
  const [products, setProducts] = useState<ProductListItem[]>([])
  const [total, setTotal] = useState(0)
  const [saleProducts, setSaleProducts] = useState<ProductListItem[]>([])
  const [category, setCategory] = useState<Category | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)
    setError(null)
    setNotFound(false)

    async function run() {
      try {
        let categoryId: string | undefined
        if (slug) {
          const cat = await api.getCategoryBySlug(slug)
          if (cancelled) return
          setCategory(cat)
          categoryId = cat.id
        } else {
          setCategory(null)
        }
        const res = await api.listProducts({
          categoryId,
          q: rawQuery || undefined,
          pageSize: 60,
        })
        if (cancelled) return
        setProducts(res.products)
        setTotal(res.total)
      } catch (e) {
        if (cancelled) return
        if (e instanceof ApiError && e.status === 404 && slug) {
          setNotFound(true)
        } else {
          setError(e instanceof Error ? e.message : 'Failed to load products.')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    run()
    return () => {
      cancelled = true
    }
  }, [slug, rawQuery])

  // Home view only: fetch the biggest discounts to drive the carousel + Mega
  // Sale row. Separate from the main list so it doesn't depend on the catalogue.
  const isHome = !slug && !rawQuery
  useEffect(() => {
    if (!isHome) return
    let cancelled = false
    api
      .listProducts({ onSale: true, sort: 'discount', pageSize: 20 })
      .then((res) => {
        if (!cancelled) setSaleProducts(res.products)
      })
      .catch(() => {
        if (!cancelled) setSaleProducts([])
      })
    return () => {
      cancelled = true
    }
  }, [isHome])

  const mode: Mode = slug && category
    ? { kind: 'category', category }
    : rawQuery
      ? { kind: 'search', query: rawQuery }
      : { kind: 'all' }

  const seo =
    mode.kind === 'category' ? (
      <Seo
        title={`${mode.category.name} — Stora`}
        description={`Shop ${mode.category.name.toLowerCase()} at Stora. Browse a wide range of products with customer reviews, clear pricing, great deals, and fast, secure checkout.`}
      />
    ) : mode.kind === 'search' ? (
      <Seo
        title={`Search: ${mode.query} — Stora`}
        description={`Search results for “${mode.query}” at Stora. Compare products, read customer reviews, check prices and deals, and check out quickly and securely.`}
      />
    ) : (
      <Seo
        title="Electronics, Furniture, Shoes & More"
        description="Shop electronics, furniture, beauty, shoes and more at Stora. Thousands of products, real customer reviews, daily deals, and fast, secure checkout."
      />
    )

  async function handleAdd(product: ProductListItem) {
    setBusyId(product.id)
    try {
      await addItem(product.id, 1)
      showToast(`${product.name} added to cart.`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not add to cart.')
    } finally {
      setBusyId(null)
    }
  }

  if (notFound) {
    return (
      <Page>
        <Seo title="Category not found" noindex />
        <Masthead eyebrow="Category" title="Category not found" />
        <p className="text-sm text-ink-soft">
          That category doesn't exist.{' '}
          <Link
            to="/"
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
          >
            Browse all products
          </Link>
        </p>
      </Page>
    )
  }

  const mastheadContent = (() => {
    if (mode.kind === 'category') {
      return (
        <Masthead
          eyebrow="Category"
          title={mode.category.name}
          caption={
            loading ? undefined : (
              <>
                <span className="tnum">{total}</span>{' '}
                {total === 1 ? 'product' : 'products'} in {mode.category.name.toLowerCase()}
              </>
            )
          }
        />
      )
    }
    if (mode.kind === 'search') {
      return (
        <Masthead
          eyebrow="Search results"
          title={<>Results for &ldquo;{mode.query}&rdquo;</>}
          caption={
            loading ? undefined : total === 0 ? (
              <>No results found</>
            ) : (
              <>
                <span className="tnum">{total}</span>{' '}
                {total === 1 ? 'result' : 'results'}
              </>
            )
          }
        />
      )
    }
    return (
      <Masthead
        title="All products"
        caption={
          loading ? undefined : (
            <>
              <span className="tnum">{total}</span>{' '}
              {total === 1 ? 'product' : 'products'}
            </>
          )
        }
      />
    )
  })()

  const masthead = (
    <>
      {seo}
      {mastheadContent}
    </>
  )

  if (loading) {
    return (
      <Page>
        {masthead}
        <p className="text-sm text-ink-soft">Loading.</p>
      </Page>
    )
  }

  if (error && products.length === 0) {
    return (
      <Page>
        {masthead}
        <p className="text-sm text-accent">{error}</p>
      </Page>
    )
  }

  if (products.length === 0) {
    return (
      <Page>
        {masthead}
        <p className="text-sm text-ink-soft">
          No products found.{' '}
          <Link
            to="/"
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
          >
            Browse all products
          </Link>
        </p>
      </Page>
    )
  }

  return (
    <Page>
      {mode.kind === 'all' && saleProducts.length > 0 && (
        <>
          <div className="mb-8 lg:mb-10">
            <PromoCarousel products={saleProducts.slice(0, 6)} />
          </div>
          <MegaSale products={saleProducts} />
        </>
      )}

      {masthead}

      {error && <p className="text-sm text-accent mb-6">{error}</p>}

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 md:gap-5">
        {products.map((p) => (
          <ProductCard
            key={p.id}
            product={p}
            busy={busyId === p.id}
            onAdd={() => handleAdd(p)}
          />
        ))}
      </div>
    </Page>
  )
}
