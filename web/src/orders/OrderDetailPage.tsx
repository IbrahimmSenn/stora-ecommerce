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

  // After Stripe redirects to /confirmation, the webhook may not have landed
  // yet — poll for ~10s while the order is still pending_payment so the page
  // flips to "Payment received" without a manual refresh.
  useEffect(() => {
    if (!id || !data) return
    if (!isConfirmation) return
    if (data.order.status !== 'pending_payment') return

    let cancelled = false
    let attempts = 0
    const tick = () => {
      if (cancelled) return
      attempts++
      api
        .getOrder(id)
        .then((d) => {
          if (cancelled) return
          setData(d)
          if (d.order.status === 'pending_payment' && attempts < 5) {
            setTimeout(tick, 2000)
          }
        })
        .catch(() => {
          // Swallow — the page already shows the previous data; user can retry manually.
        })
    }
    const t = setTimeout(tick, 2000)
    return () => {
      cancelled = true
      clearTimeout(t)
    }
  }, [id, data, isConfirmation])

  async function handleCancel() {
    if (!id || !data) return
    const wasPaid = data.order.status === 'paid'
    const prompt = wasPaid
      ? 'Cancel this order? Your payment will be refunded and stock will be restored.'
      : 'Cancel this order? Stock will be restored.'
    if (!window.confirm(prompt)) return
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
  const retryable = order.status === 'payment_failed'

  return (
    <div className="max-w-4xl mx-auto px-6 py-12">
      {isConfirmation ? (
        <header className="mb-12">
          <p className="text-xs uppercase tracking-widest text-gray-500">
            {order.status === 'paid' ? 'Payment received' : 'Order placed'}
          </p>
          <h1 className="text-4xl font-semibold mt-2">Thank you.</h1>
          {order.status === 'paid' ? (
            <p className="text-gray-600 mt-3 max-w-xl">
              Your payment went through. We'll email a receipt shortly.
            </p>
          ) : order.status === 'pending_payment' ? (
            <p className="text-gray-600 mt-3 max-w-xl">
              Stripe is still confirming your payment. This page will refresh
              once the webhook lands — usually within a few seconds.
            </p>
          ) : order.status === 'payment_failed' ? (
            <p className="text-gray-600 mt-3 max-w-xl">
              Payment didn't go through.{' '}
              <Link to={`/orders/${order.id}/pay`} className="underline">
                Try again
              </Link>
              .
            </p>
          ) : (
            <p className="text-gray-600 mt-3 max-w-xl">
              We've received your order.
            </p>
          )}
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

      {retryable && (
        <div className="border-t pt-8 mb-8">
          <Link
            to={`/orders/${order.id}/pay`}
            className="inline-block px-6 py-2 bg-gray-900 text-white text-sm uppercase tracking-wider"
          >
            Retry payment
          </Link>
          <p className="mt-2 text-xs text-gray-500">
            Your card was declined. You can try again with a different card —
            stock is held until you cancel.
          </p>
        </div>
      )}

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
            {order.status === 'paid'
              ? 'Cancelling refunds your payment via Stripe and restores stock. Once the order ships, this option goes away.'
              : 'Cancellation restores stock. Once payment is processed and the order ships, this option goes away.'}
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
