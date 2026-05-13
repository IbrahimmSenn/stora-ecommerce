import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import type { StripeElementsOptions } from '@stripe/stripe-js'
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js'

import { api, ApiError, formatPrice } from '../lib/api'
import type { OrderResponse } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useTheme } from '../lib/theme'
import { getStripe, stripeAppearance } from './stripe'
import { mapPaymentError, mapStripeError } from './errors'

export function PaymentPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { theme } = useTheme()

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
      appearance: stripeAppearance(theme),
    }
  }, [clientSecret, theme])

  if (loading) {
    return (
      <Page width="max-w-6xl">
        <Masthead number="04" eyebrow="Payment" title="Preparing." />
      </Page>
    )
  }

  if (error || !order || !clientSecret || !pk || !options) {
    return (
      <Page width="max-w-4xl">
        <Masthead
          number="04"
          eyebrow="Payment"
          title="Couldn't start."
          caption={error ?? 'Could not start payment.'}
        />
        <Link
          to="/cart"
          className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          Back to cart.
        </Link>
      </Page>
    )
  }

  return (
    <Page width="max-w-6xl">
      <Masthead
        number="04"
        eyebrow="Payment"
        title="Complete your order."
        caption={
          <>
            Order <span className="tnum">{order.order.order_number}</span>
            <span aria-hidden className="mx-2 text-rule-strong">/</span>
            total <span className="tnum">{formatPrice(order.order.total_cents)}</span>.
          </>
        }
      />

      <div className="grid lg:grid-cols-[1.4fr_1fr] gap-12 lg:gap-20">
        <Elements stripe={getStripe(pk)} options={options} key={theme}>
          <PayForm orderId={order.order.id} />
        </Elements>

        <aside className="lg:sticky lg:top-8 lg:self-start">
          <p className="uc-tight text-[0.7rem] text-ink-faint mb-4">Summary</p>
          <ul className="divide-y divide-rule border-y border-rule">
            {order.items.map((it) => (
              <li
                key={it.id}
                className="flex justify-between gap-4 py-3 text-sm"
              >
                <span className="flex-1 text-ink">
                  {it.product_name}
                  <span className="text-ink-faint"> × {it.quantity}</span>
                </span>
                <span className="tnum text-ink">
                  {formatPrice(it.unit_price_cents * it.quantity)}
                </span>
              </li>
            ))}
          </ul>
          <dl className="mt-6 space-y-2 text-sm">
            <Row label="Subtotal" value={formatPrice(order.order.subtotal_cents)} />
            <Row label="Shipping" value={formatPrice(order.order.shipping_cents)} />
            <div className="flex justify-between pt-4 mt-2 border-t border-rule items-baseline">
              <dt className="uc-tight text-[0.7rem] text-ink-faint">Total</dt>
              <dd
                className="font-display tnum text-ink text-[clamp(1.5rem,3vw,2rem)] leading-none font-bold"
              >
                {formatPrice(order.order.total_cents)}
              </dd>
            </div>
          </dl>
          <p className="mt-6 text-xs text-ink-faint leading-relaxed">
            Stripe test card{' '}
            <span className="tnum text-ink-soft">4242 4242 4242 4242</span>, any
            future date, any CVC.{' '}
            <span className="tnum text-ink-soft">4000 0000 0000 9995</span>{' '}
            simulates an insufficient-funds decline.
          </p>
        </aside>
      </div>
    </Page>
  )
}

function PayForm({ orderId }: { orderId: string }) {
  const stripe = useStripe()
  const elements = useElements()
  const location = useLocation()
  const carried =
    typeof (location.state as { error?: unknown } | null)?.error === 'string'
      ? ((location.state as { error: string }).error)
      : null
  const [submitting, setSubmitting] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(carried)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!stripe || !elements) return
    setSubmitting(true)
    setErrorMsg(null)

    try {
      const result = await stripe.confirmPayment({
        elements,
        confirmParams: {
          return_url: `${window.location.origin}/orders/${orderId}/confirmation`,
        },
      })

      // confirmPayment redirects on success. We only get here if there's an
      // immediate error (validation, decline, etc.) — Stripe never redirects
      // those, so we render the mapped message inline.
      if (result.error) {
        setErrorMsg(mapStripeError(result.error))
      }
    } catch (e) {
      setErrorMsg(mapPaymentError(e))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="max-w-xl">
      {errorMsg && (
        <p
          role="alert"
          className="mb-6 text-sm text-accent border-l-2 border-accent pl-3 py-1"
        >
          {errorMsg}
        </p>
      )}
      <PaymentElement />
      <button
        type="submit"
        disabled={!stripe || submitting}
        className="mt-8 bg-accent text-on-accent hover:bg-accent-soft transition-colors px-7 py-3 text-sm tracking-[0.01em] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
      >
        {submitting ? 'Processing.' : 'Pay now'}
      </button>
    </form>
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
