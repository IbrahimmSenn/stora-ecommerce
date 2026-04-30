import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderResponse } from '../lib/api'
import { StatusBadge, formatStatus } from './OrderStatus'

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const location = useLocation()
  const isConfirmation = location.pathname.endsWith('/confirmation')

  const [data, setData] = useState<OrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cancelling, setCancelling] = useState(false)

  useEffect(() => {
    if (!id) return
    let cancelled = false
    setLoading(true)
    api
      .getOrder(id)
      .then((d) => {
        if (!cancelled) setData(d)
      })
      .catch((e) => {
        if (cancelled) return
        if (e instanceof ApiError) setError(e.message)
        else setError('Failed to load order.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  async function handleCancel() {
    if (!id || !data) return
    if (!window.confirm('Cancel this order? Stock will be restored.')) return
    setCancelling(true)
    setError(null)
    try {
      const next = await api.cancelOrder(id)
      setData(next)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to cancel order.')
    } finally {
      setCancelling(false)
    }
  }

  if (loading) return <p className="p-8">Loading order…</p>
  if (error)
    return (
      <div className="max-w-2xl mx-auto p-8">
        <p className="text-red-600">{error}</p>
        <Link to="/orders" className="underline text-sm mt-4 inline-block">
          ← Back to orders
        </Link>
      </div>
    )
  if (!data) return null

  const { order, items, address } = data
  const cancellable = order.status === 'pending_payment' || order.status === 'paid'

  return (
    <div className="max-w-4xl mx-auto px-6 py-12">
      {isConfirmation ? (
        <header className="mb-12">
          <p className="text-xs uppercase tracking-widest text-gray-500">
            Order placed
          </p>
          <h1 className="text-4xl font-semibold mt-2">Thank you.</h1>
          <p className="text-gray-600 mt-3 max-w-xl">
            We've received your order. Payment is collected on the next milestone —
            for now your order sits in pending payment.
          </p>
        </header>
      ) : (
        <header className="mb-10">
          <Link
            to="/orders"
            className="text-xs uppercase tracking-widest text-gray-500 hover:underline"
          >
            ← All orders
          </Link>
          <h1 className="text-3xl font-semibold mt-3">Order details</h1>
        </header>
      )}

      <div className="grid sm:grid-cols-[auto_1fr] gap-x-12 gap-y-6 text-sm mb-12">
        <Detail label="Order number" value={order.order_number} mono />
        <Detail label="Placed" value={new Date(order.created_at).toLocaleString()} />
        <Detail label="Status" value={<StatusBadge status={order.status} />} />
        <Detail
          label="Shipping"
          value={`${order.shipping_method.charAt(0).toUpperCase()}${order.shipping_method.slice(1)}`}
        />
        <Detail label="Email" value={order.email} />
        {order.phone && <Detail label="Phone" value={order.phone} />}
      </div>

      <section className="mb-12">
        <h2 className="text-xs uppercase tracking-widest text-gray-500 mb-3">Items</h2>
        <ul className="divide-y border-t border-b">
          {items.map((it) => (
            <li key={it.id} className="flex justify-between gap-4 py-4">
              <div className="flex-1">
                <p className="font-medium">{it.product_name}</p>
                <p className="text-sm text-gray-500 tabular-nums">
                  {formatPrice(it.unit_price_cents)} × {it.quantity}
                </p>
              </div>
              <span className="tabular-nums">
                {formatPrice(it.unit_price_cents * it.quantity)}
              </span>
            </li>
          ))}
        </ul>
        <dl className="mt-6 ml-auto max-w-xs space-y-2 text-sm">
          <Row label="Subtotal" value={formatPrice(order.subtotal_cents)} />
          <Row label="Shipping" value={formatPrice(order.shipping_cents)} />
          <div className="flex justify-between pt-3 border-t font-semibold text-base">
            <dt>Total</dt>
            <dd className="tabular-nums">{formatPrice(order.total_cents)}</dd>
          </div>
        </dl>
      </section>

      <section className="mb-12">
        <h2 className="text-xs uppercase tracking-widest text-gray-500 mb-3">
          Shipping to
        </h2>
        <address className="not-italic text-sm leading-relaxed">
          {address.recipient_name}
          <br />
          {address.line1}
          {address.line2 && (
            <>
              <br />
              {address.line2}
            </>
          )}
          <br />
          {address.city}, {address.region} {address.postal_code}
          <br />
          {address.country}
        </address>
      </section>

      {cancellable && (
        <div className="border-t pt-8">
          <button
            type="button"
            onClick={handleCancel}
            disabled={cancelling}
            className="text-sm text-red-700 hover:underline disabled:opacity-50"
          >
            {cancelling ? 'Cancelling…' : `Cancel this order (${formatStatus(order.status)})`}
          </button>
          <p className="mt-2 text-xs text-gray-500">
            Cancellation restores stock. Once payment is processed and the order
            ships, this option goes away.
          </p>
        </div>
      )}
    </div>
  )
}

function Detail({
  label,
  value,
  mono,
}: {
  label: string
  value: React.ReactNode
  mono?: boolean
}) {
  return (
    <>
      <dt className="text-gray-500 uppercase text-xs tracking-widest">{label}</dt>
      <dd className={mono ? 'tabular-nums' : ''}>{value}</dd>
    </>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <dt className="text-gray-500">{label}</dt>
      <dd className="tabular-nums">{value}</dd>
    </div>
  )
}
