import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { loadStripe } from '@stripe/stripe-js'
import type { Stripe, StripeElementsOptions } from '@stripe/stripe-js'
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js'

import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderResponse } from '../lib/api'

// Cache Stripe per publishable key — loadStripe should fire once per
// pageload (per Stripe's own docs).
const stripePromiseCache = new Map<string, Promise<Stripe | null>>()
function getStripe(pk: string) {
  let p = stripePromiseCache.get(pk)
  if (!p) {
    p = loadStripe(pk)
    stripePromiseCache.set(pk, p)
  }
  return p
}

export function PaymentPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [order, setOrder] = useState<OrderResponse | null>(null)
  const [clientSecret, setClientSecret] = useState<string | null>(null)
  const [pk, setPk] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    let cancelled = false

    async function init() {
      try {
        if (!id) return
        const o = await api.getOrder(id)
        if (cancelled) return

        // If the order is already in a terminal state, the pay page makes no sense.
        if (
          o.order.status !== 'pending_payment' &&
          o.order.status !== 'payment_failed'
        ) {
          navigate(`/orders/${id}/confirmation`, { replace: true })
          return
        }

        setOrder(o)

        const intent = await api.createPaymentIntent(id)
        if (cancelled) return
        setClientSecret(intent.client_secret)
        setPk(intent.publishable_key)
      } catch (e) {
        if (cancelled) return
        if (e instanceof ApiError) setError(e.message)
        else setError('Failed to start payment.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void init()
    return () => {
      cancelled = true
    }
  }, [id, navigate])

  const options = useMemo<StripeElementsOptions | null>(() => {
    if (!clientSecret) return null
    return {
      clientSecret,
      appearance: {
        theme: 'flat',
        variables: {
          colorPrimary: '#111111',
          fontFamily: 'system-ui, -apple-system, sans-serif',
          borderRadius: '0px',
        },
      },
    }
  }, [clientSecret])

  if (loading) return <p className="p-8">Preparing payment…</p>

  if (error || !order || !clientSecret || !pk || !options) {
    return (
      <div className="max-w-2xl mx-auto p-8">
        <p className="text-red-600">{error ?? 'Could not start payment.'}</p>
        <Link to="/cart" className="underline text-sm mt-4 inline-block">
          ← Back to cart
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-6xl mx-auto px-6 py-12">
      <header className="mb-12">
        <p className="text-xs uppercase tracking-widest text-gray-500">Payment</p>
        <h1 className="text-4xl font-semibold mt-2">Complete your order</h1>
        <p className="text-gray-600 mt-3 max-w-xl">
          Order <span className="tabular-nums">{order.order.order_number}</span> ·
          total <span className="tabular-nums">{formatPrice(order.order.total_cents)}</span>
        </p>
      </header>

      <div className="grid lg:grid-cols-[1.4fr_1fr] gap-16">
        <Elements stripe={getStripe(pk)} options={options}>
          <PayForm orderId={order.order.id} />
        </Elements>

        <aside className="lg:sticky lg:top-8 lg:self-start">
          <p className="text-xs uppercase tracking-widest text-gray-500 mb-4">Summary</p>
          <ul className="divide-y border-t border-b">
            {order.items.map((it) => (
              <li key={it.id} className="flex justify-between gap-4 py-3 text-sm">
                <span className="flex-1">
                  {it.product_name}
                  <span className="text-gray-500"> × {it.quantity}</span>
                </span>
                <span className="tabular-nums">
                  {formatPrice(it.unit_price_cents * it.quantity)}
                </span>
              </li>
            ))}
          </ul>
          <dl className="mt-6 space-y-2 text-sm">
            <Row label="Subtotal" value={formatPrice(order.order.subtotal_cents)} />
            <Row label="Shipping" value={formatPrice(order.order.shipping_cents)} />
            <div className="flex justify-between pt-3 border-t text-base font-semibold">
              <dt>Total</dt>
              <dd className="tabular-nums">{formatPrice(order.order.total_cents)}</dd>
            </div>
          </dl>
          <p className="mt-6 text-xs text-gray-500">
            Use Stripe test card <code>4242 4242 4242 4242</code>, any future
            date, any CVC. <code>4000 0000 0000 9995</code> simulates an
            insufficient-funds decline.
          </p>
        </aside>
      </div>
    </div>
  )
}

function PayForm({ orderId }: { orderId: string }) {
  const stripe = useStripe()
  const elements = useElements()
  const [submitting, setSubmitting] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!stripe || !elements) return
    setSubmitting(true)
    setErrorMsg(null)

    const result = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${window.location.origin}/orders/${orderId}/confirmation`,
      },
    })

    // confirmPayment redirects on success. We only get here if there's an
    // immediate error (validation, decline, etc.) — Stripe never redirects
    // those, so we render the message inline.
    if (result.error) {
      setErrorMsg(result.error.message ?? 'Payment failed.')
    }
    setSubmitting(false)
  }

  return (
    <form onSubmit={handleSubmit} className="max-w-xl">
      {errorMsg && (
        <div
          role="alert"
          className="mb-6 px-4 py-3 border border-red-200 bg-red-50 text-red-800 text-sm"
        >
          {errorMsg}
        </div>
      )}
      <PaymentElement />
      <button
        type="submit"
        disabled={!stripe || submitting}
        className="mt-8 w-full sm:w-auto px-8 py-3 bg-gray-900 text-white text-sm uppercase tracking-wider disabled:opacity-50"
      >
        {submitting ? 'Processing…' : 'Pay now'}
      </button>
    </form>
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
