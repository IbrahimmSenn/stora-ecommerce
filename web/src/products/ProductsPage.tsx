import { useEffect, useState } from 'react'
import { api, formatPrice } from '../lib/api'
import type { ProductListItem } from '../lib/api'
import { useCart } from '../cart/useCart'

export function ProductsPage() {
  const { addItem } = useCart()
  const [products, setProducts] = useState<ProductListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<string | null>(null)

  useEffect(() => {
    api
      .listProducts()
      .then((r) => setProducts(r.products))
      .catch((e) => setError(e instanceof Error ? e.message : 'failed to load'))
      .finally(() => setLoading(false))
  }, [])

  async function handleAdd(id: string, name: string) {
    setBusyId(id)
    setFeedback(null)
    try {
      await addItem(id, 1)
      setFeedback(`Added "${name}" to cart.`)
    } catch (e) {
      setFeedback(e instanceof Error ? e.message : 'could not add to cart')
    } finally {
      setBusyId(null)
    }
  }

  if (loading) return <p className="p-8">Loading products…</p>
  if (error) return <p className="p-8 text-red-600">{error}</p>

  return (
    <div className="max-w-5xl mx-auto p-8">
      <h1 className="text-2xl font-semibold mb-6">Shop</h1>

      {feedback && (
        <p role="status" className="mb-4 text-sm text-gray-700">
          {feedback}
        </p>
      )}

      <ul className="grid sm:grid-cols-2 md:grid-cols-3 gap-4">
        {products.map((p) => (
          <li key={p.id} className="border rounded p-4 flex flex-col">
            {p.primary_image ? (
              <img
                src={p.primary_image}
                alt=""
                className="aspect-square object-cover mb-3 bg-gray-100"
              />
            ) : (
              <div className="aspect-square bg-gray-100 mb-3" />
            )}
            <p className="font-medium">{p.name}</p>
            {p.brand_name && (
              <p className="text-sm text-gray-500">{p.brand_name}</p>
            )}
            <p className="mt-2 tabular-nums">{formatPrice(p.price)}</p>
            <p className="text-xs text-gray-500 mb-3">
              {p.stock_quantity > 0
                ? `${p.stock_quantity} in stock`
                : 'Out of stock'}
            </p>
            <button
              type="button"
              disabled={busyId === p.id || p.stock_quantity === 0}
              onClick={() => handleAdd(p.id, p.name)}
              className="mt-auto bg-black text-white py-2 rounded disabled:opacity-50"
            >
              {busyId === p.id ? 'Adding…' : 'Add to cart'}
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
