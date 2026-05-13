import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api, ApiError, formatPrice } from '../lib/api'
import type { Category, ProductListItem } from '../lib/api'
import { useCart } from '../cart/useCart'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Plus } from '../components/icons'
import { useToast } from '../components/useToast'

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
        })
        if (cancelled) return
        setProducts(res.products)
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

  const mode: Mode = slug && category
    ? { kind: 'category', category }
    : rawQuery
      ? { kind: 'search', query: rawQuery }
      : { kind: 'all' }

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
        <Masthead number="01" eyebrow="Category" title="Not found." />
        <p className="text-sm text-ink-soft">
          That category doesn't exist.{' '}
          <Link
            to="/"
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
          >
            Browse all.
          </Link>
        </p>
      </Page>
    )
  }

  const masthead = (() => {
    if (mode.kind === 'category') {
      return (
        <Masthead
          number="01"
          eyebrow="Category"
          title={`${mode.category.name}.`}
          caption={
            loading ? undefined : (
              <>
                <span className="tnum">{products.length}</span>{' '}
                {products.length === 1 ? 'item' : 'items'} in {mode.category.name.toLowerCase()}.
              </>
            )
          }
        />
      )
    }
    if (mode.kind === 'search') {
      return (
        <Masthead
          number="01"
          eyebrow="Search"
          title={<>&ldquo;{mode.query}&rdquo;.</>}
          caption={
            loading ? undefined : products.length === 0 ? (
              <>No matches.</>
            ) : (
              <>
                <span className="tnum">{products.length}</span>{' '}
                {products.length === 1 ? 'match' : 'matches'}.
              </>
            )
          }
        />
      )
    }
    return (
      <Masthead
        number="01"
        eyebrow="Catalogue"
        title="Shop."
        caption={
          loading ? undefined : (
            <>
              A small, considered selection.{' '}
              <span className="tnum">{products.length}</span>{' '}
              {products.length === 1 ? 'item' : 'items'} in stock.
            </>
          )
        }
      />
    )
  })()

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
          Nothing here.{' '}
          <Link
            to="/"
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
          >
            Browse all.
          </Link>
        </p>
      </Page>
    )
  }

  // Featured layout only on the unfiltered home view — a "Featured" badge on a
  // filtered subset would lie about the selection.
  const showFeatured = mode.kind === 'all'
  const featured: ProductListItem | null = showFeatured ? products[0] : null
  const rest: ProductListItem[] = showFeatured ? products.slice(1) : products

  return (
    <Page>
      {masthead}

      {error && <p className="text-sm text-accent mb-6">{error}</p>}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-x-10 gap-y-16 lg:gap-x-16 lg:gap-y-24">
        {featured && (
          <article className="md:col-span-2 md:row-span-2 flex flex-col gap-6 h-full group">
            <Link
              to={`/product/${featured.id}`}
              aria-label={featured.name}
              className="contents"
            >
              <div className="relative">
                <span className="absolute -left-2 -top-6 uc-tight text-[0.7rem] text-ink-faint">
                  <span className="tnum">F</span>
                  <span aria-hidden className="text-rule-strong mx-2">/</span>
                  Featured
                </span>
                <div className="aspect-[4/5] bg-sunken overflow-hidden">
                  {featured.primary_image ? (
                    <img
                      src={featured.primary_image}
                      alt=""
                      loading="eager"
                      className="w-full h-full object-cover"
                    />
                  ) : null}
                </div>
              </div>
            </Link>
            <div className="mt-auto flex flex-col gap-6">
              <Link
                to={`/product/${featured.id}`}
                className="flex items-end justify-between gap-6 group/title"
              >
                <div className="min-w-0">
                  <h2
                    className="font-display text-[clamp(1.5rem,3vw,2.25rem)] leading-tight text-ink font-bold group-hover/title:text-accent transition-colors"
                  >
                    {featured.name}
                  </h2>
                  {featured.brand_name && (
                    <p className="text-sm text-ink-soft mt-1">{featured.brand_name}</p>
                  )}
                </div>
                <p className="tnum text-ink text-lg shrink-0">
                  {formatPrice(featured.price)}
                </p>
              </Link>
              <div className="flex items-center justify-between">
                <StockSignal qty={featured.stock_quantity} />
                <QuickAddButton
                  productName={featured.name}
                  busy={busyId === featured.id}
                  disabled={featured.stock_quantity === 0}
                  onClick={() => handleAdd(featured)}
                />
              </div>
            </div>
          </article>
        )}

        {rest.map((p) => (
          <article key={p.id} className="flex flex-col gap-4 h-full">
            <Link
              to={`/product/${p.id}`}
              aria-label={p.name}
              className="contents group/card"
            >
              <div className="aspect-square bg-sunken overflow-hidden p-[8%]">
                {p.primary_image ? (
                  <img
                    src={p.primary_image}
                    alt=""
                    loading="lazy"
                    className="w-full h-full object-contain"
                  />
                ) : null}
              </div>
              <div>
                <h3 className="text-ink text-[0.95rem] leading-snug group-hover/card:text-accent transition-colors">
                  {p.name}
                </h3>
                {p.brand_name && (
                  <p className="text-xs text-ink-faint mt-0.5">{p.brand_name}</p>
                )}
              </div>
            </Link>
            <div className="mt-auto flex items-end justify-between gap-4">
              <div className="min-w-0">
                <p className="tnum text-ink text-sm">{formatPrice(p.price)}</p>
                <div className="mt-1">
                  <StockSignal qty={p.stock_quantity} />
                </div>
              </div>
              <QuickAddButton
                productName={p.name}
                busy={busyId === p.id}
                disabled={p.stock_quantity === 0}
                onClick={() => handleAdd(p)}
              />
            </div>
          </article>
        ))}
      </div>
    </Page>
  )
}
