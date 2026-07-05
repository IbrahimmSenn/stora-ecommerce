import { useCallback, useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api, ApiError, formatPrice, discountPercent } from '../lib/api'
import type { Brand, Category, ProductListItem } from '../lib/api'
import { useCart } from '../cart/useCart'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import {
  Plus,
  LayoutGrid,
  List as ListIcon,
  SlidersHorizontal,
  ChevronLeft,
  ChevronRight,
} from '../components/icons'
import { useToast } from '../components/useToast'
import { StarRating } from '../reviews/StarRating'
import { Seo } from '../components/Seo'
import { PromoCarousel } from './PromoCarousel'
import { MegaSale } from './MegaSale'
import { ProductGridSkeleton } from '../components/Skeleton'

const PAGE_SIZE = 24

const SORT_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'Featured' },
  { value: 'price_asc', label: 'Price: low to high' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'rating', label: 'Top rated' },
  { value: 'discount', label: 'Biggest discount' },
]

const RATING_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'Any rating' },
  { value: '4', label: '4 stars & up' },
  { value: '3', label: '3 stars & up' },
  { value: '2', label: '2 stars & up' },
]

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

function ProductRow({
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
    <article className="group flex gap-4 sm:gap-6 rounded-lg border border-rule bg-raised p-3 transition-shadow hover:border-rule-strong hover:shadow-[0_6px_20px_oklch(0.2_0.01_265/0.10)]">
      <Link
        to={`/product/${product.id}`}
        aria-label={product.name}
        className="relative h-24 w-24 sm:h-28 sm:w-28 shrink-0 bg-surface rounded-md overflow-hidden p-[6%]"
      >
        {onSale && (
          <span className="absolute left-1 top-1 z-10 rounded bg-accent px-1.5 py-0.5 text-[0.7rem] font-semibold text-on-accent tnum">
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
          {onSale ? (
            <>
              <span className="block tnum text-accent text-lg font-bold">
                {formatPrice(product.sale_price!)}
              </span>
              <span className="tnum text-ink-faint line-through text-xs">
                {formatPrice(product.price)}
              </span>
            </>
          ) : (
            <p className="tnum text-ink text-lg font-bold">{formatPrice(product.price)}</p>
          )}
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
  const [searchParams, setSearchParams] = useSearchParams()
  const rawQuery = searchParams.get('q')?.trim() ?? ''

  // Filter / sort / paging state lives in the URL so it's shareable and
  // survives the back button.
  const sort = searchParams.get('sort') ?? ''
  const brand = searchParams.get('brand') ?? ''
  const catFilter = searchParams.get('cat') ?? ''
  const minParam = searchParams.get('min') ?? ''
  const maxParam = searchParams.get('max') ?? ''
  const rating = searchParams.get('rating') ?? ''
  const page = Math.max(1, parseInt(searchParams.get('page') ?? '1', 10) || 1)
  const view: 'grid' | 'list' = searchParams.get('view') === 'list' ? 'list' : 'grid'

  const { addItem } = useCart()
  const { show: showToast } = useToast()
  const [products, setProducts] = useState<ProductListItem[]>([])
  const [total, setTotal] = useState(0)
  const [saleProducts, setSaleProducts] = useState<ProductListItem[]>([])
  const [category, setCategory] = useState<Category | null>(null)
  const [brands, setBrands] = useState<Brand[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [showFilters, setShowFilters] = useState(false)

  const updateParams = useCallback(
    (updates: Record<string, string | null>, resetPage = true) => {
      const next = new URLSearchParams(searchParams)
      for (const [k, v] of Object.entries(updates)) {
        if (v === null || v === '') next.delete(k)
        else next.set(k, v)
      }
      if (resetPage && !('page' in updates)) next.delete('page')
      setSearchParams(next)
    },
    [searchParams, setSearchParams],
  )

  // Filter selects are populated from the catalogue's brands + top-level
  // categories. Fetched once; failures just leave the filter empty.
  useEffect(() => {
    let cancelled = false
    Promise.allSettled([api.listBrands(), api.listCategories()]).then(([b, c]) => {
      if (cancelled) return
      if (b.status === 'fulfilled') setBrands(b.value)
      if (c.status === 'fulfilled') setCategories(c.value)
    })
    return () => {
      cancelled = true
    }
  }, [])

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
          categoryId = catFilter || undefined
        }
        const res = await api.listProducts({
          categoryId,
          q: rawQuery || undefined,
          page,
          pageSize: PAGE_SIZE,
          sort: sort || undefined,
          brandId: brand || undefined,
          minPrice: minParam ? Math.round(parseFloat(minParam) * 100) : undefined,
          maxPrice: maxParam ? Math.round(parseFloat(maxParam) * 100) : undefined,
          minRating: rating ? parseInt(rating, 10) : undefined,
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
  }, [slug, rawQuery, page, sort, brand, catFilter, minParam, maxParam, rating])

  const activeFilters = [brand, catFilter, minParam, maxParam, rating].filter(Boolean).length
  const pristineHome =
    !slug && !rawQuery && page === 1 && !sort && activeFilters === 0

  // Home view only: fetch the biggest discounts to drive the carousel + Mega
  // Sale row. Separate from the main list so it doesn't depend on the catalogue.
  useEffect(() => {
    if (!pristineHome) return
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
  }, [pristineHome])

  const mode: Mode = slug && category
    ? { kind: 'category', category }
    : rawQuery
      ? { kind: 'search', query: rawQuery }
      : { kind: 'all' }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function goToPage(p: number) {
    updateParams({ page: p <= 1 ? null : String(p) }, false)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function clearFilters() {
    updateParams({ brand: null, cat: null, min: null, max: null, rating: null })
  }

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

  const countLabel = mode.kind === 'search' ? (total === 1 ? 'result' : 'results') : (total === 1 ? 'product' : 'products')
  const mastheadContent =
    mode.kind === 'category' ? (
      <Masthead
        eyebrow="Category"
        title={mode.category.name}
        caption={loading ? undefined : (
          <><span className="tnum">{total}</span> {countLabel} in {mode.category.name.toLowerCase()}</>
        )}
      />
    ) : mode.kind === 'search' ? (
      <Masthead
        eyebrow="Search results"
        title={<>Results for &ldquo;{mode.query}&rdquo;</>}
        caption={loading ? undefined : total === 0 ? <>No results found</> : (
          <><span className="tnum">{total}</span> {countLabel}</>
        )}
      />
    ) : (
      <Masthead
        title="All products"
        caption={loading ? undefined : (<><span className="tnum">{total}</span> {countLabel}</>)}
      />
    )

  return (
    <Page>
      {pristineHome && saleProducts.length > 0 && (
        <>
          <div className="mb-8 lg:mb-10">
            <PromoCarousel products={saleProducts.slice(0, 6)} />
          </div>
          <MegaSale products={saleProducts} />
        </>
      )}

      {seo}
      {mastheadContent}

      {/* Controls: filters toggle, sort, grid/list view */}
      <div className="flex flex-wrap items-center gap-3 border-y border-rule py-3 mb-6">
        <button
          type="button"
          onClick={() => setShowFilters((s) => !s)}
          aria-expanded={showFilters}
          className="inline-flex items-center gap-2 text-sm text-ink-soft hover:text-ink transition-colors cursor-pointer"
        >
          <SlidersHorizontal size={16} strokeWidth={1.5} aria-hidden />
          Filters
          {activeFilters > 0 && (
            <span className="tnum inline-flex h-5 min-w-5 items-center justify-center bg-accent px-1 text-[0.7rem] text-on-accent">
              {activeFilters}
            </span>
          )}
        </button>

        <div className="ms-auto flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-ink-soft">
            <span className="uc-tight text-[0.7rem] text-ink-faint">Sort</span>
            <select
              value={sort}
              onChange={(e) => updateParams({ sort: e.target.value || null })}
              className="bg-raised border border-rule px-2 py-1.5 text-ink cursor-pointer focus:border-accent outline-none"
              style={{ borderRadius: 0 }}
            >
              {SORT_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </label>

          <div className="flex items-center border border-rule" role="group" aria-label="View">
            <button
              type="button"
              onClick={() => updateParams({ view: null }, false)}
              aria-pressed={view === 'grid'}
              aria-label="Grid view"
              className={`inline-flex h-8 w-8 items-center justify-center cursor-pointer transition-colors ${view === 'grid' ? 'bg-ink text-on-accent' : 'text-ink-soft hover:text-ink'}`}
            >
              <LayoutGrid size={16} strokeWidth={1.5} aria-hidden />
            </button>
            <button
              type="button"
              onClick={() => updateParams({ view: 'list' }, false)}
              aria-pressed={view === 'list'}
              aria-label="List view"
              className={`inline-flex h-8 w-8 items-center justify-center cursor-pointer transition-colors ${view === 'list' ? 'bg-ink text-on-accent' : 'text-ink-soft hover:text-ink'}`}
            >
              <ListIcon size={16} strokeWidth={1.5} aria-hidden />
            </button>
          </div>
        </div>
      </div>

      {showFilters && (
        <FilterPanel
          brands={brands}
          categories={categories}
          showCategory={mode.kind !== 'category'}
          brand={brand}
          cat={catFilter}
          min={minParam}
          max={maxParam}
          rating={rating}
          activeFilters={activeFilters}
          onChange={updateParams}
          onClear={clearFilters}
        />
      )}

      {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}

      {loading ? (
        <ProductGridSkeleton count={12} />
      ) : products.length === 0 ? (
        <p className="text-sm text-ink-soft">
          No products match these filters.{' '}
          {activeFilters > 0 && (
            <button type="button" onClick={clearFilters} className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors cursor-pointer">
              Clear filters
            </button>
          )}
        </p>
      ) : view === 'list' ? (
        <div className="flex flex-col gap-3">
          {products.map((p) => (
            <ProductRow key={p.id} product={p} busy={busyId === p.id} onAdd={() => handleAdd(p)} />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 md:gap-5">
          {products.map((p) => (
            <ProductCard key={p.id} product={p} busy={busyId === p.id} onAdd={() => handleAdd(p)} />
          ))}
        </div>
      )}

      {!loading && totalPages > 1 && (
        <Pager page={page} totalPages={totalPages} onGo={goToPage} />
      )}
    </Page>
  )
}

function FilterPanel({
  brands,
  categories,
  showCategory,
  brand,
  cat,
  min,
  max,
  rating,
  activeFilters,
  onChange,
  onClear,
}: {
  brands: Brand[]
  categories: Category[]
  showCategory: boolean
  brand: string
  cat: string
  min: string
  max: string
  rating: string
  activeFilters: number
  onChange: (updates: Record<string, string | null>) => void
  onClear: () => void
}) {
  const [minInput, setMinInput] = useState(min)
  const [maxInput, setMaxInput] = useState(max)

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setMinInput(min)
    setMaxInput(max)
  }, [min, max])

  function applyPrice(e: React.FormEvent) {
    e.preventDefault()
    onChange({ min: minInput || null, max: maxInput || null })
  }

  return (
    <div className="border border-rule bg-raised p-4 mb-6 flex flex-wrap items-end gap-x-6 gap-y-4">
      {showCategory && (
        <label className="flex flex-col gap-1.5">
          <span className="uc-tight text-[0.7rem] text-ink-faint">Category</span>
          <select
            value={cat}
            onChange={(e) => onChange({ cat: e.target.value || null })}
            className="bg-surface border border-rule px-2 py-1.5 text-sm text-ink cursor-pointer focus:border-accent outline-none min-w-[10rem]"
            style={{ borderRadius: 0 }}
          >
            <option value="">All categories</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
        </label>
      )}

      <label className="flex flex-col gap-1.5">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Brand</span>
        <select
          value={brand}
          onChange={(e) => onChange({ brand: e.target.value || null })}
          className="bg-surface border border-rule px-2 py-1.5 text-sm text-ink cursor-pointer focus:border-accent outline-none min-w-[10rem]"
          style={{ borderRadius: 0 }}
        >
          <option value="">All brands</option>
          {brands.map((b) => (
            <option key={b.id} value={b.id}>{b.name}</option>
          ))}
        </select>
      </label>

      <label className="flex flex-col gap-1.5">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Rating</span>
        <select
          value={rating}
          onChange={(e) => onChange({ rating: e.target.value || null })}
          className="bg-surface border border-rule px-2 py-1.5 text-sm text-ink cursor-pointer focus:border-accent outline-none"
          style={{ borderRadius: 0 }}
        >
          {RATING_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      </label>

      <form onSubmit={applyPrice} className="flex items-end gap-2">
        <label className="flex flex-col gap-1.5">
          <span className="uc-tight text-[0.7rem] text-ink-faint">Min $</span>
          <input
            type="number"
            min="0"
            inputMode="decimal"
            value={minInput}
            onChange={(e) => setMinInput(e.target.value)}
            onBlur={applyPrice}
            className="bg-surface border border-rule px-2 py-1.5 text-sm text-ink w-20 focus:border-accent outline-none"
            style={{ borderRadius: 0 }}
          />
        </label>
        <label className="flex flex-col gap-1.5">
          <span className="uc-tight text-[0.7rem] text-ink-faint">Max $</span>
          <input
            type="number"
            min="0"
            inputMode="decimal"
            value={maxInput}
            onChange={(e) => setMaxInput(e.target.value)}
            onBlur={applyPrice}
            className="bg-surface border border-rule px-2 py-1.5 text-sm text-ink w-20 focus:border-accent outline-none"
            style={{ borderRadius: 0 }}
          />
        </label>
        <button type="submit" className="border border-rule px-3 py-1.5 text-sm text-ink-soft hover:border-ink hover:text-ink transition-colors cursor-pointer">
          Apply
        </button>
      </form>

      {activeFilters > 0 && (
        <button
          type="button"
          onClick={onClear}
          className="text-sm text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer ms-auto"
        >
          Clear all
        </button>
      )}
    </div>
  )
}

function Pager({ page, totalPages, onGo }: { page: number; totalPages: number; onGo: (p: number) => void }) {
  // Windowed page numbers around the current page.
  const window = 2
  const start = Math.max(1, page - window)
  const end = Math.min(totalPages, page + window)
  const pages: number[] = []
  for (let i = start; i <= end; i++) pages.push(i)

  return (
    <nav aria-label="Pagination" className="flex items-center justify-center gap-1 mt-10">
      <button
        type="button"
        onClick={() => onGo(page - 1)}
        disabled={page <= 1}
        aria-label="Previous page"
        className="inline-flex h-9 w-9 items-center justify-center border border-rule text-ink-soft hover:border-ink hover:text-ink transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
      >
        <ChevronLeft size={16} strokeWidth={1.5} aria-hidden />
      </button>

      {start > 1 && (
        <>
          <PageButton n={1} current={page} onGo={onGo} />
          {start > 2 && <span className="px-1 text-ink-faint">…</span>}
        </>
      )}

      {pages.map((p) => (
        <PageButton key={p} n={p} current={page} onGo={onGo} />
      ))}

      {end < totalPages && (
        <>
          {end < totalPages - 1 && <span className="px-1 text-ink-faint">…</span>}
          <PageButton n={totalPages} current={page} onGo={onGo} />
        </>
      )}

      <button
        type="button"
        onClick={() => onGo(page + 1)}
        disabled={page >= totalPages}
        aria-label="Next page"
        className="inline-flex h-9 w-9 items-center justify-center border border-rule text-ink-soft hover:border-ink hover:text-ink transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
      >
        <ChevronRight size={16} strokeWidth={1.5} aria-hidden />
      </button>
    </nav>
  )
}

function PageButton({ n, current, onGo }: { n: number; current: number; onGo: (p: number) => void }) {
  const active = n === current
  return (
    <button
      type="button"
      onClick={() => onGo(n)}
      aria-current={active ? 'page' : undefined}
      className={`inline-flex h-9 min-w-9 items-center justify-center border px-2 tnum text-sm transition-colors cursor-pointer ${
        active ? 'border-ink bg-ink text-on-accent' : 'border-rule text-ink-soft hover:border-ink hover:text-ink'
      }`}
    >
      {n}
    </button>
  )
}
