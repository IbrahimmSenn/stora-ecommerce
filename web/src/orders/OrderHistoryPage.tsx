import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderSummary } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useAuth } from '../auth/useAuth'
import { StatusBadge } from './OrderStatus'

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'All' },
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
  const { initializing } = useAuth()
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
    // Wait for the auth-context mount-refresh to settle so the API sees the
    // user, not a guest. See OrderDetailPage for the same gate.
    if (initializing) return
    void load()
  }, [load, initializing])

  return (
    <Page width="max-w-4xl">
      <Masthead
        eyebrow="Account"
        title="Orders."
        caption="Every order you've placed, with current status and totals."
      />

      <div className="flex flex-wrap gap-2 mb-6 border-b border-rule pb-4">
        {STATUS_FILTERS.map((f) => {
          const active = status === f.value
          return (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setStatus(f.value)}
              className={`text-xs px-3 py-1.5 transition-colors cursor-pointer ${
                active
                  ? 'bg-ink text-on-accent'
                  : 'text-ink-soft hover:text-ink'
              }`}
              style={{ borderRadius: 0 }}
            >
              {f.label}
            </button>
          )
        })}
      </div>

      <div className="grid grid-cols-2 gap-6 mb-10 max-w-md">
        <label className="block">
          <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
            From
          </span>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors"
            style={{ borderRadius: 0 }}
          />
        </label>
        <label className="block">
          <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
            To
          </span>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors"
            style={{ borderRadius: 0 }}
          />
        </label>
      </div>

      {loading && <p className="text-sm text-ink-soft">Loading.</p>}

      {error && (
        <p role="alert" className="text-sm text-accent">
          {error}
        </p>
      )}

      {!loading && !error && orders && orders.length === 0 && (
        <div className="border-t border-rule pt-12">
          <p className="text-ink-soft mb-3">No orders match these filters.</p>
          <Link
            to="/"
            className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent"
          >
            Back to the shop.
          </Link>
        </div>
      )}

      {!loading && orders && orders.length > 0 && (
        <ul className="divide-y divide-rule border-y border-rule">
          {orders.map((o) => (
            <li key={o.id}>
              <Link
                to={`/orders/${o.id}`}
                className="grid grid-cols-[7rem_1fr_auto_auto] items-baseline gap-6 py-5 group"
              >
                <span className="tnum text-sm text-ink group-hover:text-accent transition-colors">
                  {o.order_number}
                </span>
                <span className="text-xs text-ink-faint">
                  {new Date(o.created_at).toLocaleDateString()}
                  <span aria-hidden className="mx-2 text-rule-strong">/</span>
                  <span className="tnum">{o.item_count}</span>{' '}
                  {o.item_count === 1 ? 'item' : 'items'}
                </span>
                <StatusBadge status={o.status} />
                <span className="tnum text-ink min-w-[5rem] text-right">
                  {formatPrice(o.total_cents)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Page>
  )
}
