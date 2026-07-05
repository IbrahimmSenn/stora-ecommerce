import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderResponse } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useAuth } from '../auth/useAuth'
import { StatusBadge, formatStatus } from './OrderStatus'
import { OrderItemReview } from './OrderItemReview'

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const location = useLocation()
  const isConfirmation = location.pathname.endsWith('/confirmation')
  const { initializing } = useAuth()

  const [data, setData] = useState<OrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cancelling, setCancelling] = useState(false)
  const [etaByCode, setEtaByCode] = useState<Record<string, string>>({})

  // Delivery ETA labels are admin-managed; load them once to translate the
  // order's shipping_method code into a human estimate.
  useEffect(() => {
    let cancelled = false
    api
      .listDeliveryOptions()
      .then((opts) => {
        if (cancelled) return
        setEtaByCode(Object.fromEntries(opts.map((o) => [o.code, o.eta_label])))
      })
      .catch(() => {
        // Non-fatal — we just won't show an ETA.
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!id) return
    // Wait for the AuthProvider's mount-time refresh to settle before
    // fetching — otherwise a page reload (e.g. the Stripe checkout redirect)
    // hits the API unauthenticated and gets ErrForbidden.
    if (initializing) return
    let cancelled = false
    // eslint-disable-next-line react-hooks/set-state-in-effect
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
  }, [id, initializing])

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

  if (loading) {
    return (
      <Page width="max-w-4xl">
        <Masthead eyebrow="Order" title="Loading." />
      </Page>
    )
  }

  if (error) {
    return (
      <Page width="max-w-4xl">
        <Masthead eyebrow="Order" title="Couldn't load." caption={error} />
        <Link
          to="/orders"
          className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          Back to all orders.
        </Link>
      </Page>
    )
  }

  if (!data) return null

  const { order, items, address } = data
  const cancellable = order.status === 'pending_payment' || order.status === 'paid'
  const retryable = order.status === 'payment_failed'
  const settled = ['paid', 'processing', 'shipped', 'delivered'].includes(order.status)
  const reviewableItems = items.filter((it) => it.product_id)
  const showReviewPrompt = isConfirmation && settled && reviewableItems.length > 0

  return (
    <Page width="max-w-4xl">
      {isConfirmation ? (
        <Masthead
          eyebrow={order.status === 'paid' ? 'Payment received' : 'Order placed'}
          title="Thank you."
          caption={
            order.status === 'paid'
              ? "Your payment went through. We'll email a receipt shortly."
              : order.status === 'pending_payment'
                ? 'Stripe is still confirming your payment. This page will refresh once the webhook lands — usually within a few seconds.'
                : order.status === 'payment_failed' ? (
                    <>
                      Payment didn't go through.{' '}
                      <Link
                        to={`/orders/${order.id}/pay`}
                        className="text-ink underline underline-offset-4 decoration-accent"
                      >
                        Try again.
                      </Link>
                    </>
                  )
                : "We've received your order."
          }
        />
      ) : (
        <>
          <Link
            to="/orders"
            className="uc-tight text-[0.7rem] text-ink-faint hover:text-ink transition-colors inline-block mb-6"
          >
            ← All orders
          </Link>
          <Masthead eyebrow="Order" title={order.order_number} />
        </>
      )}

      <dl className="grid sm:grid-cols-[auto_1fr] gap-x-12 gap-y-4 text-sm mb-14">
        <Detail label="Order number" value={order.order_number} mono />
        <Detail
          label="Placed"
          value={new Date(order.created_at).toLocaleString()}
        />
        <Detail label="Status" value={<StatusBadge status={order.status} />} />
        <Detail
          label="Shipping"
          value={`${order.shipping_method.charAt(0).toUpperCase()}${order.shipping_method.slice(1)}`}
        />
        {etaByCode[order.shipping_method] && settled && (
          <Detail label="Estimated delivery" value={etaByCode[order.shipping_method]} />
        )}
        <Detail label="Email" value={order.email} />
        {order.phone && <Detail label="Phone" value={order.phone} />}
      </dl>

      <section className="mb-14">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-4">
          Items
        </h2>
        <ul className="divide-y divide-rule border-y border-rule">
          {items.map((it) => (
            <li key={it.id} className="flex items-center gap-4 py-5">
              {it.product_id ? (
                <Link to={`/product/${it.product_id}`} className="shrink-0" title={it.product_name}>
                  <ItemThumb url={it.thumbnail_url} name={it.product_name} />
                </Link>
              ) : (
                <ItemThumb url={it.thumbnail_url} name={it.product_name} />
              )}
              <div className="flex-1 min-w-0">
                {it.product_id ? (
                  <Link
                    to={`/product/${it.product_id}`}
                    className="text-ink hover:text-accent transition-colors underline-offset-4 hover:underline decoration-rule-strong"
                  >
                    {it.product_name}
                  </Link>
                ) : (
                  <p className="text-ink">{it.product_name}</p>
                )}
                <p className="text-xs text-ink-faint tnum mt-1">
                  {formatPrice(it.unit_price_cents)} × {it.quantity}
                </p>
              </div>
              <span className="tnum text-ink shrink-0">
                {formatPrice(it.unit_price_cents * it.quantity)}
              </span>
            </li>
          ))}
        </ul>
        <dl className="mt-6 ml-auto max-w-xs space-y-2 text-sm">
          <Row label="Subtotal" value={formatPrice(order.subtotal_cents)} />
          <Row label="Shipping" value={formatPrice(order.shipping_cents)} />
          <div className="flex justify-between pt-4 mt-2 border-t border-rule items-baseline">
            <dt className="uc-tight text-[0.7rem] text-ink-faint">Total</dt>
            <dd
              className="font-display tnum text-ink text-[clamp(1.25rem,2.5vw,1.75rem)] leading-none font-bold"
            >
              {formatPrice(order.total_cents)}
            </dd>
          </div>
        </dl>
      </section>

      <section className="mb-14">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-4">
          Shipping to
        </h2>
        <address className="not-italic text-sm leading-relaxed text-ink">
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

      {showReviewPrompt && (
        <section className="border-t border-rule pt-8 mb-14">
          <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-1">Rate your purchase</h2>
          <p className="text-sm text-ink-soft mb-4">
            Bought it — now tell other shoppers what you think. Reviews go live right away.
          </p>
          <ul className="divide-y divide-rule border-y border-rule">
            {reviewableItems.map((it) => (
              <OrderItemReview key={it.id} item={it} />
            ))}
          </ul>
        </section>
      )}

      {retryable && (
        <div className="border-t border-rule pt-8 mb-8">
          <Link
            to={`/orders/${order.id}/pay`}
            className="inline-block bg-accent text-on-accent hover:bg-accent-soft transition-colors px-6 py-3 text-sm tracking-[0.01em]"
          >
            Retry payment
          </Link>
          <p className="mt-3 text-xs text-ink-faint max-w-md">
            Your card was declined. You can try again with a different card —
            stock is held until you cancel.
          </p>
        </div>
      )}

      {cancellable && (
        <div className="border-t border-rule pt-8">
          <button
            type="button"
            onClick={handleCancel}
            disabled={cancelling}
            className="text-sm text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {cancelling
              ? 'Cancelling.'
              : `Cancel this order (${formatStatus(order.status).toLowerCase()}).`}
          </button>
          <p className="mt-3 text-xs text-ink-faint max-w-md">
            {order.status === 'paid'
              ? 'Cancelling refunds your payment via Stripe and restores stock. Once the order ships, this option goes away.'
              : 'Cancellation restores stock. Once payment is processed and the order ships, this option goes away.'}
          </p>
        </div>
      )}
    </Page>
  )
}

function ItemThumb({ url, name }: { url?: string | null; name: string }) {
  return (
    <span className="block h-14 w-14 shrink-0 overflow-hidden bg-sunken" style={{ borderRadius: 0 }}>
      {url ? (
        <img src={url} alt={name} className="h-full w-full object-cover" loading="lazy" />
      ) : (
        <span aria-hidden className="block h-full w-full" />
      )}
    </span>
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
      <dt className="uc-tight text-[0.7rem] text-ink-faint self-baseline">
        {label}
      </dt>
      <dd className={`text-ink ${mono ? 'tnum' : ''}`}>{value}</dd>
    </>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <dt className="text-ink-soft">{label}</dt>
      <dd className="tnum text-ink">{value}</dd>
    </div>
  )
}
