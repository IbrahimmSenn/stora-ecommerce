import { useCart } from './useCart'
import { formatPrice } from '../lib/api'
import { Link } from 'react-router-dom'

export function CartPage() {
  const { cart, loading, error, updateItem, removeItem, clear } = useCart()

  if (loading) return <p className="p-8">Loading cart…</p>

  if (error) return <p className="p-8 text-red-600">{error}</p>

  if (!cart || cart.items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto p-8 text-center">
        <h1 className="text-2xl font-semibold mb-2">Your cart is empty</h1>
        <p className="text-gray-600 mb-4">Nothing here yet.</p>
        <Link to="/" className="underline">
          Browse products
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto p-8">
      <h1 className="text-2xl font-semibold mb-6">Your cart</h1>

      <ul className="divide-y border rounded">
        {cart.items.map((it) => (
          <li key={it.id} className="flex items-center gap-4 p-4">
            <div className="flex-1">
              <p className="font-medium">{it.product_name}</p>
              <p className="text-sm text-gray-500 tabular-nums">
                {formatPrice(it.product_price)} each · {it.stock} in stock
              </p>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() =>
                  it.quantity > 1 && updateItem(it.product_id, it.quantity - 1)
                }
                disabled={it.quantity <= 1}
                className="w-8 h-8 border rounded disabled:opacity-30"
                aria-label="Decrease quantity"
              >
                −
              </button>
              <span className="w-8 text-center tabular-nums">{it.quantity}</span>
              <button
                type="button"
                onClick={() =>
                  it.quantity < it.stock &&
                  updateItem(it.product_id, it.quantity + 1)
                }
                disabled={it.quantity >= it.stock}
                className="w-8 h-8 border rounded disabled:opacity-30"
                aria-label="Increase quantity"
              >
                +
              </button>
            </div>
            <div className="w-20 text-right tabular-nums">
              {formatPrice(it.product_price * it.quantity)}
            </div>
            <button
              type="button"
              onClick={() => removeItem(it.product_id)}
              className="text-sm text-red-600 hover:underline"
            >
              Remove
            </button>
          </li>
        ))}
      </ul>

      <div className="flex justify-between items-center mt-6">
        <button
          type="button"
          onClick={() => clear()}
          className="text-sm text-gray-600 hover:underline"
        >
          Clear cart
        </button>
        <div className="text-right">
          <p className="text-sm text-gray-500">Total</p>
          <p className="text-2xl font-semibold tabular-nums">
            {formatPrice(cart.total)}
          </p>
        </div>
      </div>

      <div className="mt-8 flex justify-end">
        <Link
          to="/checkout"
          className="px-8 py-3 bg-gray-900 text-white text-sm uppercase tracking-wider"
        >
          Checkout
        </Link>
      </div>
    </div>
  )
}
