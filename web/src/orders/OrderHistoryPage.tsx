import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderSummary } from '../lib/api'
import { StatusBadge } from './OrderStatus'

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'All statuses' },
  { value: 'pending_payment', label: 'Pending payment' },
  { value: 'paid', label: 'Paid' },
  { value: 'payment_failed', label: 'Payment failed' },
  { value: 'processing', label: 'Processing' },
  { value: 'shipped', label: 'Shipped' },
  { value: 'delivered', label: 'Delivered' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'refunded', label: 'Refunded' },
]

export function OrderHistoryPage() {
  const [orders, setOrders] = useState<OrderSummary[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [status, setStatus] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params: { status?: string; from?: string; to?: string } = {}
      if (status) params.status = status
      if (from) params.from = new Date(from).toISOString()
      if (to) {
        const d = new Date(to)
        d.setHours(23, 59, 59, 999)
        params.to = d.toISOString()
      }
      const list = await api.listOrders(params)
      setOrders(list)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to load orders.')
    } finally {
      setLoading(false)
    }
  }, [status, from, to])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div className="max-w-4xl mx-auto px-6 py-12">
      <header className="mb-10">
        <p className="text-xs uppercase tracking-widest text-gray-500">Account</p>
        <h1 className="text-3xl font-semibold mt-2">Your orders</h1>
      </header>

      <div className="grid sm:grid-cols-[1fr_1fr_1fr] gap-4 mb-8">
        <label className="block text-sm">
          <span className="block text-gray-500 mb-1 text-xs uppercase tracking-widest">
            Status
          </span>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="w-full border border-gray-300 px-3 py-2"
          >
            {STATUS_FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="block text-gray-500 mb-1 text-xs uppercase tracking-widest">
            From
          </span>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="w-full border border-gray-300 px-3 py-2"
          />
        </label>
        <label className="block text-sm">
          <span className="block text-gray-500 mb-1 text-xs uppercase tracking-widest">
            To
          </span>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="w-full border border-gray-300 px-3 py-2"
          />
        </label>
      </div>

      {loading && <p className="text-sm text-gray-500">Loading…</p>}

      {error && (
        <p
          role="alert"
          className="px-4 py-3 border border-red-200 bg-red-50 text-red-800 text-sm"
        >
          {error}
        </p>
      )}

      {!loading && !error && orders && orders.length === 0 && (
        <div className="border-t pt-12 text-center">
          <p className="text-gray-600 mb-4">No orders match these filters.</p>
          <Link to="/" className="underline">
            Browse products
          </Link>
        </div>
      )}

      {!loading && orders && orders.length > 0 && (
        <ul className="divide-y border-t border-b">
          {orders.map((o) => (
            <li key={o.id}>
              <Link
                to={`/orders/${o.id}`}
                className="grid grid-cols-[auto_1fr_auto_auto] items-center gap-6 py-4 hover:bg-gray-50 px-2 -mx-2"
              >
                <span className="tabular-nums text-sm">{o.order_number}</span>
                <span className="text-sm text-gray-500">
                  {new Date(o.created_at).toLocaleDateString()} ·{' '}
                  {o.item_count} {o.item_count === 1 ? 'item' : 'items'}
                </span>
                <StatusBadge status={o.status} />
                <span className="tabular-nums font-medium">
                  {formatPrice(o.total_cents)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
