import { Link, useLocation } from 'react-router-dom'
import { useCart } from './useCart'
import { formatPrice } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { RecommendationsRail } from './RecommendationsRail'

function Notice({ text }: { text: string }) {
  return (
    <p
      role="status"
      className="mb-8 text-sm text-accent border-l-2 border-accent pl-3 py-1"
    >
      {text}
    </p>
  )
}

function QtyButton({
  onClick,
  disabled,
  label,
  children,
}: {
  onClick: () => void
  disabled: boolean
  label: string
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      className="w-7 h-7 border border-rule-strong text-ink hover:border-ink hover:text-accent transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
      style={{ borderRadius: 0 }}
    >
      {children}
    </button>
  )
}

function Thumb({
  url,
  name,
  size = 'h-16 w-16',
}: {
  url?: string | null
  name: string
  size?: string
}) {
  return (
    <div className={`${size} bg-sunken shrink-0 overflow-hidden`} aria-hidden>
      {url ? (
        <img
          src={url}
          alt=""
          loading="lazy"
          className="w-full h-full object-cover"
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center px-1">
          <span className="text-[0.55rem] text-ink-faint uc-tight text-center leading-tight line-clamp-2">
            {name}
          </span>
        </div>
      )}
    </div>
  )
}

export function CartPage() {
  const { cart, loading, error, updateItem, removeItem, clear } = useCart()
  const location = useLocation()
  const notice =
    typeof (location.state as { notice?: unknown } | null)?.notice === 'string'
      ? ((location.state as { notice: string }).notice)
      : null

  if (loading) {
    return (
      <Page width="max-w-4xl">
        <Masthead eyebrow="Cart" title="Your cart" />
        <p className="text-sm text-ink-soft">Loading.</p>
      </Page>
    )
  }

  if (error) {
    return (
      <Page width="max-w-4xl">
        <Masthead eyebrow="Cart" title="Your cart" />
        <p className="text-sm text-accent">{error}</p>
      </Page>
    )
  }

  if (!cart || cart.items.length === 0) {
    return (
      <Page width="max-w-4xl">
        <Masthead
          eyebrow="Cart"
          title="Your cart is empty"
          caption="Your cart is empty. Browse products and add items to get started."
        />
        {notice && <Notice text={notice} />}
        <Link
          to="/"
          className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          Continue shopping
        </Link>
      </Page>
    )
  }

  return (
    <Page width="max-w-4xl">
      <Masthead
        eyebrow="Cart"
        title="Your cart"
        caption={
          <>
            <span className="tnum">{cart.items.length}</span>{' '}
            {cart.items.length === 1 ? 'item' : 'items'} in your cart. Adjust
            quantities, remove items, or proceed to checkout.
          </>
        }
      />
      {notice && <Notice text={notice} />}

      <div className="grid grid-cols-1 lg:grid-cols-[1fr_auto] gap-x-16 gap-y-12">
        <ul className="divide-y divide-rule border-y border-rule">
          {cart.items.map((it) => (
            <li
              key={it.id}
              className="grid grid-cols-[4rem_1fr_auto_auto] items-center gap-x-6 gap-y-2 py-6"
            >
              <Thumb url={it.image_url} name={it.product_name} />
              <div className="min-w-0">
                <p className="text-ink leading-snug">{it.product_name}</p>
                <p className="text-xs text-ink-faint mt-1 tnum">
                  {formatPrice(it.product_price)} each · {it.stock} in stock
                </p>
              </div>
              <div className="flex items-center gap-2 justify-self-end">
                <QtyButton
                  onClick={() =>
                    it.quantity > 1 && updateItem(it.product_id, it.quantity - 1)
                  }
                  disabled={it.quantity <= 1}
                  label="Decrease quantity"
                >
                  −
                </QtyButton>
                <span className="w-8 text-center tnum text-ink">{it.quantity}</span>
                <QtyButton
                  onClick={() =>
                    it.quantity < it.stock &&
                    updateItem(it.product_id, it.quantity + 1)
                  }
                  disabled={it.quantity >= it.stock}
                  label="Increase quantity"
                >
                  +
                </QtyButton>
              </div>
              <div className="tnum text-ink text-right justify-self-end min-w-[5rem]">
                {formatPrice(it.product_price * it.quantity)}
              </div>
              <button
                type="button"
                onClick={() => removeItem(it.product_id)}
                className="col-start-2 col-span-3 justify-self-end text-xs text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer"
              >
                Remove
              </button>
            </li>
          ))}
        </ul>

        <aside className="lg:sticky lg:top-8 lg:self-start lg:w-64 flex flex-col gap-8">
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">
              Subtotal
            </p>
            <p
              className="font-display tnum text-ink text-[clamp(2rem,4vw,2.75rem)] leading-none font-bold"
            >
              {formatPrice(cart.total)}
            </p>
            <p className="text-xs text-ink-faint mt-3">
              Shipping calculated at checkout.
            </p>
          </div>

          <div className="flex flex-col gap-3">
            <Link
              to="/checkout"
              className="bg-accent text-on-accent hover:bg-accent-soft transition-colors px-5 py-3 text-sm tracking-[0.01em] text-center"
            >
              Proceed to checkout
            </Link>
            <Link
              to="/"
              className="text-sm text-ink-soft hover:text-ink underline underline-offset-4 text-center"
            >
              Keep shopping
            </Link>
          </div>

          <div className="pt-6 border-t border-rule">
            <Button variant="link" onClick={() => clear()}>
              Clear cart.
            </Button>
          </div>
        </aside>
      </div>

      <RecommendationsRail cartVersion={cart.items.length} />
    </Page>
  )
}
