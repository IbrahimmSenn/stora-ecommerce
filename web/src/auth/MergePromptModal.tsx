import { useState } from 'react'
import type { Cart } from '../lib/api'
import { formatPrice } from '../lib/api'
import { useCart } from '../cart/useCart'

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
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-10">
      <div className="bg-white rounded-lg max-w-2xl w-full p-6">
        <h2 className="text-xl font-semibold mb-2">Two carts found</h2>
        <p className="text-sm text-gray-600 mb-6">
          You added items as a guest and already have items in your account. Pick which cart to keep — the other will be discarded.
        </p>

        <div className="grid md:grid-cols-2 gap-4 mb-6">
          <CartSummary title="Your guest cart" cart={guestCart} />
          <CartSummary title="Your account cart" cart={userCart} />
        </div>

        {error && <p className="text-sm text-red-600 mb-3">{error}</p>}

        <div className="flex justify-end gap-3">
          <button
            type="button"
            disabled={busy}
            onClick={() => choose('user')}
            className="px-4 py-2 border rounded disabled:opacity-50"
          >
            Use account cart
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => choose('guest')}
            className="px-4 py-2 bg-black text-white rounded disabled:opacity-50"
          >
            Use guest cart
          </button>
        </div>
      </div>
    </div>
  )
}

function CartSummary({ title, cart }: { title: string; cart: Cart }) {
  return (
    <div className="border rounded p-3">
      <h3 className="font-medium mb-2">{title}</h3>
      {cart.items.length === 0 ? (
        <p className="text-sm text-gray-500">Empty</p>
      ) : (
        <ul className="space-y-1 text-sm">
          {cart.items.map((it) => (
            <li key={it.id} className="flex justify-between">
              <span>
                {it.product_name} × {it.quantity}
              </span>
              <span className="tabular-nums">
                {formatPrice(it.product_price * it.quantity)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <div className="mt-2 pt-2 border-t text-sm flex justify-between font-medium">
        <span>Total</span>
        <span className="tabular-nums">{formatPrice(cart.total)}</span>
      </div>
    </div>
  )
}
