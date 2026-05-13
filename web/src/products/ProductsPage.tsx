import { useEffect, useState } from 'react'
import { api, formatPrice } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { useCart } from '../cart/useCart'
import { useCartPanel } from '../cart/useCartPanel'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'

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

function AddButton({
  busy,
  added,
  disabled,
  onClick,
}: {
  busy: boolean
  added: boolean
  disabled: boolean
  onClick: (el: HTMLButtonElement | null) => void
}) {
  return (
    <button
      type="button"
      onClick={(e) => onClick(e.currentTarget)}
      disabled={disabled || busy}
      className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:no-underline"
    >
      {busy ? 'Adding.' : added ? 'Added.' : disabled ? 'Unavailable' : 'Add to cart'}
    </button>
  )
}

export function ProductsPage() {
  const { addItem } = useCart()
  const { openWith } = useCartPanel()
  const [products, setProducts] = useState<ProductListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [recentlyAdded, setRecentlyAdded] = useState<string | null>(null)

  useEffect(() => {
    api
      .listProducts()
      .then((r) => setProducts(r.products))
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load products.'))
      .finally(() => setLoading(false))
  }, [])

  async function handleAdd(product: ProductListItem, trigger: HTMLButtonElement | null) {
    setBusyId(product.id)
    try {
      await addItem(product.id, 1)
      setRecentlyAdded(product.id)
      openWith(
        {
          productId: product.id,
          productName: product.name,
          unitPriceCents: product.price,
          quantity: 1,
          imageUrl: product.primary_image ?? null,
        },
        trigger,
      )
      setTimeout(
        () => setRecentlyAdded((curr) => (curr === product.id ? null : curr)),
        1800,
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not add to cart.')
    } finally {
      setBusyId(null)
    }
  }

  if (loading) {
    return (
      <Page>
        <Masthead number="01" eyebrow="Catalogue" title="Shop." />
        <p className="text-sm text-ink-soft">Loading.</p>
      </Page>
    )
  }

  if (error && products.length === 0) {
    return (
      <Page>
        <Masthead number="01" eyebrow="Catalogue" title="Shop." />
        <p className="text-sm text-accent">{error}</p>
      </Page>
    )
  }

  const [featured, ...rest] = products

  return (
    <Page>
      <Masthead
        number="01"
        eyebrow="Catalogue"
        title="Shop."
        caption={
          <>
            A small, considered selection.{' '}
            <span className="tnum">{products.length}</span>{' '}
            {products.length === 1 ? 'item' : 'items'} in stock.
          </>
        }
      />

      {error && <p className="text-sm text-accent mb-6">{error}</p>}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-x-10 gap-y-16 lg:gap-x-16 lg:gap-y-24">
        {featured && (
          <article className="md:col-span-2 md:row-span-2 flex flex-col gap-6">
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
            <div className="flex items-end justify-between gap-6">
              <div className="min-w-0">
                <h2
                  className="font-display text-[clamp(1.5rem,3vw,2.25rem)] leading-tight text-ink font-bold"
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
            </div>
            <div className="flex items-baseline justify-between">
              <StockSignal qty={featured.stock_quantity} />
              <AddButton
                busy={busyId === featured.id}
                added={recentlyAdded === featured.id}
                disabled={featured.stock_quantity === 0}
                onClick={(el) => handleAdd(featured, el)}
              />
            </div>
          </article>
        )}

        {rest.map((p, i) => {
          const offsetTop = i % 3 === 1 ? 'md:mt-12' : i % 3 === 2 ? 'md:mt-4' : ''
          return (
            <article
              key={p.id}
              className={`flex flex-col gap-4 ${offsetTop}`}
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
                <h3 className="text-ink text-[0.95rem] leading-snug">{p.name}</h3>
                {p.brand_name && (
                  <p className="text-xs text-ink-faint mt-0.5">{p.brand_name}</p>
                )}
              </div>
              <div className="flex items-baseline justify-between">
                <p className="tnum text-ink text-sm">{formatPrice(p.price)}</p>
                <StockSignal qty={p.stock_quantity} />
              </div>
              <AddButton
                busy={busyId === p.id}
                added={recentlyAdded === p.id}
                disabled={p.stock_quantity === 0}
                onClick={(el) => handleAdd(p, el)}
              />
            </article>
          )
        })}
      </div>

      {products.length === 0 && (
        <p className="text-sm text-ink-faint">No products in the catalogue yet.</p>
      )}
    </Page>
  )
}
