import { useCallback, useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import {
  api,
  ApiError,
  formatPrice,
  type AdminOrderSummary,
  type OrderResponse,
} from '../lib/api'

const STATUS_FILTERS = [
  { value: '', label: 'All' },
  { value: 'pending_payment', label: 'Pending' },
  { value: 'paid', label: 'Paid' },
  { value: 'processing', label: 'Processing' },
  { value: 'shipped', label: 'Shipped' },
  { value: 'delivered', label: 'Delivered' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'refunded', label: 'Refunded' },
]

// Statuses an admin may set manually (mirrors the server's allow-list).
const SETTABLE = ['processing', 'shipped', 'delivered']

function statusLabel(s: string) {
  return s.replace(/_/g, ' ')
}

export function AdminOrdersPage() {
  const [orders, setOrders] = useState<AdminOrderSummary[]>([])
  const [total, setTotal] = useState(0)
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<OrderResponse | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const refresh = useCallback(() => {
    setLoading(true)
    setError(null)
    api
      .adminListOrders({ status: status || undefined })
      .then((res) => {
        setOrders(res.orders)
        setTotal(res.total)
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : 'Could not load orders.'))
      .finally(() => setLoading(false))
  }, [status])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh()
  }, [refresh])

  async function openOrder(id: string) {
    setActionMsg(null)
    try {
      setSelected(await api.adminGetOrder(id))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not load order.')
    }
  }

  async function setOrderStatus(id: string, next: string) {
    setActionMsg(null)
    try {
      const updated = await api.adminUpdateOrderStatus(id, next)
      setSelected(updated)
      setActionMsg(`Status set to ${statusLabel(next)}.`)
      refresh()
    } catch (e) {
      setActionMsg(e instanceof ApiError ? e.message : 'Could not update status.')
    }
  }

  async function refund(id: string) {
    setActionMsg(null)
    try {
      const updated = await api.adminRefundOrder(id)
      setSelected(updated)
      setActionMsg('Order refunded.')
      refresh()
    } catch (e) {
      setActionMsg(e instanceof ApiError ? e.message : 'Could not refund order.')
    }
  }

  return (
    <Page width="max-w-5xl">
      <Masthead eyebrow="Operations" title="Orders." caption="Every order across all customers." />

      <div className="flex flex-wrap items-center gap-2 mb-8">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => setStatus(f.value)}
            className={`text-xs px-3 py-1.5 border transition-colors cursor-pointer ${
              status === f.value
                ? 'border-accent text-accent'
                : 'border-rule text-ink-soft hover:border-ink hover:text-ink'
            }`}
          >
            {f.label}
          </button>
        ))}
        <span className="uc-tight text-[0.7rem] text-ink-faint ml-auto tnum">{total} total</span>
      </div>

      {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}
      {loading ? (
        <p className="text-sm text-ink-soft">Loading.</p>
      ) : orders.length === 0 ? (
        <p className="text-sm text-ink-faint">No orders match this filter.</p>
      ) : (
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b border-rule-strong text-left">
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Order</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Customer</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Status</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4 text-right">Total</th>
              <th className="py-2"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-rule">
            {orders.map((o) => (
              <tr key={o.id}>
                <td className="py-3 pr-4 tnum text-ink">{o.order_number}</td>
                <td className="py-3 pr-4 text-ink-soft">
                  {o.email}
                  {o.is_guest && <span className="uc-tight text-[0.65rem] text-ink-faint ml-2">guest</span>}
                </td>
                <td className="py-3 pr-4">
                  <span className="uc-tight text-[0.7rem] text-ink-soft">{statusLabel(o.status)}</span>
                </td>
                <td className="py-3 pr-4 text-right tnum text-ink">{formatPrice(o.total_cents)}</td>
                <td className="py-3 text-right">
                  <button
                    type="button"
                    onClick={() => openOrder(o.id)}
                    className="text-xs text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent cursor-pointer"
                  >
                    Manage
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {selected && (
        <OrderDrawer
          order={selected}
          actionMsg={actionMsg}
          onClose={() => setSelected(null)}
          onSetStatus={setOrderStatus}
          onRefund={refund}
        />
      )}
    </Page>
  )
}

function OrderDrawer({
  order,
  actionMsg,
  onClose,
  onSetStatus,
  onRefund,
}: {
  order: OrderResponse
  actionMsg: string | null
  onClose: () => void
  onSetStatus: (id: string, status: string) => void
  onRefund: (id: string) => void
}) {
  const o = order.order
  const canRefund = ['paid', 'processing', 'shipped', 'delivered'].includes(o.status)

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true" aria-label={`Order ${o.order_number}`}>
      <button type="button" aria-label="Close" className="absolute inset-0 bg-ink/16 cursor-default" onClick={onClose} />
      <div className="relative w-full max-w-md bg-surface h-full overflow-y-auto p-8 shadow-xl">
        <div className="flex items-baseline justify-between mb-6">
          <h2 className="font-display text-2xl text-ink font-bold tnum">{o.order_number}</h2>
          <button type="button" onClick={onClose} className="text-sm text-ink-faint hover:text-ink cursor-pointer">Close</button>
        </div>

        <dl className="text-sm flex flex-col gap-2 border-b border-rule pb-6 mb-6">
          <Row label="Status" value={statusLabel(o.status)} />
          <Row label="Customer" value={o.email} />
          <Row label="Placed" value={new Date(o.created_at).toLocaleString()} />
          <Row label="Shipping" value={statusLabel(o.shipping_method)} />
          <Row label="Total" value={formatPrice(o.total_cents)} />
        </dl>

        <h3 className="uc-tight text-[0.7rem] text-ink-faint mb-3">Items</h3>
        <ul className="flex flex-col gap-2 text-sm mb-6">
          {order.items.map((it) => (
            <li key={it.id} className="flex justify-between gap-4">
              <span className="text-ink-soft">{it.product_name} × {it.quantity}</span>
              <span className="tnum text-ink">{formatPrice(it.unit_price_cents * it.quantity)}</span>
            </li>
          ))}
        </ul>

        <h3 className="uc-tight text-[0.7rem] text-ink-faint mb-3">Ship to</h3>
        <address className="not-italic text-sm text-ink-soft mb-8 leading-relaxed">
          {order.address.recipient_name}<br />
          {order.address.line1}{order.address.line2 ? `, ${order.address.line2}` : ''}<br />
          {order.address.city}, {order.address.region} {order.address.postal_code}<br />
          {order.address.country}
        </address>

        <h3 className="uc-tight text-[0.7rem] text-ink-faint mb-3">Update shipping status</h3>
        <div className="flex flex-wrap gap-2 mb-6">
          {SETTABLE.map((s) => (
            <button
              key={s}
              type="button"
              disabled={o.status === s}
              onClick={() => onSetStatus(o.id, s)}
              className="text-xs px-3 py-1.5 border border-rule text-ink-soft hover:border-accent hover:text-accent transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {statusLabel(s)}
            </button>
          ))}
        </div>

        {canRefund && (
          <button
            type="button"
            onClick={() => onRefund(o.id)}
            className="text-xs px-3 py-1.5 border border-negative text-negative hover:bg-negative hover:text-on-accent transition-colors cursor-pointer"
          >
            Process refund
          </button>
        )}

        {actionMsg && <p className="text-xs text-ink-soft mt-4" role="status">{actionMsg}</p>}
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-6">
      <dt className="uc-tight text-[0.7rem] text-ink-faint">{label}</dt>
      <dd className="text-ink text-right">{value}</dd>
    </div>
  )
}
