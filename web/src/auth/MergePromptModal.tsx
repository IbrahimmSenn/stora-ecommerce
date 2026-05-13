import { useState } from 'react'
import type { Cart } from '../lib/api'
import { formatPrice } from '../lib/api'
import { useCart } from '../cart/useCart'
import { Button } from '../components/Button'

type Props = {
  guestCart: Cart
  userCart: Cart
  onResolved: () => void
}

export function MergePromptModal({ guestCart, userCart, onResolved }: Props) {
  const { resolveMerge } = useCart()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function choose(strategy: 'guest' | 'user') {
    setError(null)
    setBusy(true)
    try {
      await resolveMerge(strategy)
      onResolved()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'merge failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-ink/40 flex items-center justify-center p-4 z-10">
      <div className="bg-raised border border-rule max-w-2xl w-full p-6">
        <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">Cart</p>
        <h2 className="font-display text-xl text-ink font-bold mb-2">
          Two carts found.
        </h2>
        <p className="text-sm text-ink-soft mb-6 max-w-[55ch]">
          You added items as a guest and already have items in your account. Pick which cart to keep — the other will be discarded.
        </p>

        <div className="grid md:grid-cols-2 gap-4 mb-6">
          <CartSummary title="Your guest cart" cart={guestCart} />
          <CartSummary title="Your account cart" cart={userCart} />
        </div>

        {error && <p className="text-xs text-accent mb-3">{error}</p>}

        <div className="flex justify-end gap-3">
          <Button
            variant="ghost"
            disabled={busy}
            onClick={() => choose('user')}
          >
            Use account cart
          </Button>
          <Button
            variant="primary"
            disabled={busy}
            onClick={() => choose('guest')}
          >
            Use guest cart
          </Button>
        </div>
      </div>
    </div>
  )
}

function CartSummary({ title, cart }: { title: string; cart: Cart }) {
  return (
    <div className="border border-rule p-3">
      <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">{title}</p>
      {cart.items.length === 0 ? (
        <p className="text-sm text-ink-faint">Empty.</p>
      ) : (
        <ul className="space-y-1 text-sm text-ink">
          {cart.items.map((it) => (
            <li key={it.id} className="flex justify-between">
              <span>
                {it.product_name} × {it.quantity}
              </span>
              <span className="tnum">
                {formatPrice(it.product_price * it.quantity)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <div className="mt-2 pt-2 border-t border-rule text-sm flex justify-between text-ink">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Total</span>
        <span className="tnum">{formatPrice(cart.total)}</span>
      </div>
    </div>
  )
}
