/* CheckoutPage — single-page checkout in the Zalando/Wolt pattern.
 *
 * Contact + shipping address + shipping method + payment selection live
 * on this one route. Stripe Elements mount inline via the deferred-intent
 * pattern (mode: 'payment'); on submit we create the order, then the
 * PaymentIntent, then confirm — single user action, one "Place order" click.
 *
 * If payment confirmation fails after the order has been created, the order
 * is left in pending_payment and the user is sent to /orders/:id/pay for
 * retry. That keeps the happy path on this page and gives a clear recovery
 * route for declines and timeouts.
 */
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js'
import type { StripeElementsOptions } from '@stripe/stripe-js'
import { useCart } from '../cart/useCart'
import { useAuth } from '../auth/useAuth'
import { api, ApiError, formatPrice } from '../lib/api'
import type { CheckoutRequest, ShippingMethod } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useTheme } from '../lib/theme'
import { getStripe, stripeAppearance } from '../payment/stripe'
import { mapPaymentError, mapStripeError } from '../payment/errors'
import { validateCheckout } from './validate'
import type { CheckoutFormState } from './validate'

type FormState = CheckoutFormState

const SHIPPING_OPTIONS: { id: ShippingMethod; label: string; cents: number; eta: string }[] = [
  { id: 'standard', label: 'Standard', cents: 500, eta: '5–7 business days' },
  { id: 'express', label: 'Express', cents: 1500, eta: '1–2 business days' },
]

const initial: FormState = {
  email: '',
  phone: '',
  shipping_method: 'standard',
  recipient_name: '',
  line1: '',
  line2: '',
  city: '',
  region: '',
  postal_code: '',
  country: '',
}

export function CheckoutPage() {
  const { cart, loading } = useCart()
  const { theme } = useTheme()
  const [publishableKey, setPublishableKey] = useState<string | null>(null)
  const [configError, setConfigError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .getStripeConfig()
      .then((cfg) => {
        if (!cancelled) setPublishableKey(cfg.publishable_key)
      })
      .catch((e) => {
        if (cancelled) return
        setConfigError(
          e instanceof ApiError
            ? e.message
            : 'Could not initialise the payment provider.',
        )
      })
    return () => {
      cancelled = true
    }
  }, [])

  const shippingCents =
    SHIPPING_OPTIONS.find((o) => o.id === 'standard')!.cents
  const seedAmount = (cart?.total ?? 0) + shippingCents

  const options = useMemo<StripeElementsOptions | null>(() => {
    if (!publishableKey || seedAmount <= 0) return null
    return {
      mode: 'payment',
      amount: seedAmount,
      currency: 'usd',
      appearance: stripeAppearance(theme),
    }
  }, [publishableKey, seedAmount, theme])

  if (loading) {
    return (
      <Page width="max-w-6xl">
        <Masthead number="03" eyebrow="Checkout" title="Review." />
        <p className="text-sm text-ink-soft">Loading.</p>
      </Page>
    )
  }

  if (!cart || cart.items.length === 0) {
    return (
      <Page width="max-w-4xl">
        <Masthead
          number="03"
          eyebrow="Checkout"
          title="Nothing to check out."
          caption="Your cart is empty."
        />
        <Link
          to="/"
          className="text-sm text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          Back to the shop.
        </Link>
      </Page>
    )
  }

  if (configError || !publishableKey || !options) {
    return (
      <Page width="max-w-4xl">
        <Masthead
          number="03"
          eyebrow="Checkout"
          title="Couldn't start."
          caption={configError ?? 'Loading payment provider.'}
        />
      </Page>
    )
  }

  return (
    <Elements stripe={getStripe(publishableKey)} options={options} key={theme}>
      <CheckoutInner />
    </Elements>
  )
}

function CheckoutInner() {
  const { cart, updateItem, removeItem } = useCart()
  const { email: authedEmail, isAuthed } = useAuth()
  const navigate = useNavigate()
  const stripe = useStripe()
  const elements = useElements()
  const [lineBusy, setLineBusy] = useState<string | null>(null)
  const [lineError, setLineError] = useState<string | null>(null)

  async function adjust(productId: string, nextQty: number) {
    setLineBusy(productId)
    setLineError(null)
    try {
      if (nextQty <= 0) {
        await removeItem(productId)
      } else {
        await updateItem(productId, nextQty)
      }
    } catch (e) {
      setLineError(
        e instanceof ApiError ? e.message : "Couldn't update the cart line.",
      )
    } finally {
      setLineBusy(null)
    }
  }

  const [form, setForm] = useState<FormState>(initial)
  const [touched, setTouched] = useState<Partial<Record<keyof FormState, boolean>>>({})
  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)
  const [prefilledEmail, setPrefilledEmail] = useState(false)

  useEffect(() => {
    if (authedEmail && !form.email) {
      setForm((f) => ({ ...f, email: authedEmail }))
      setPrefilledEmail(true)
    }
  }, [authedEmail, form.email])

  const shipping = useMemo(
    () => SHIPPING_OPTIONS.find((o) => o.id === form.shipping_method)!,
    [form.shipping_method],
  )
  const subtotal = cart?.total ?? 0
  const total = subtotal + shipping.cents

  // Keep Stripe Elements' internal amount in sync when the shipping method
  // changes — required for the deferred-intent confirm step to validate.
  useEffect(() => {
    if (!elements) return
    elements.update({ amount: total })
  }, [elements, total])

  const errors = useMemo(() => validateCheckout(form), [form])
  const isValid = Object.keys(errors).length === 0

  function update<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [key]: value }))
  }
  function markTouched(key: keyof FormState) {
    setTouched((t) => ({ ...t, [key]: true }))
  }
  function err(key: keyof FormState): string | undefined {
    return touched[key] ? errors[key] : undefined
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setServerError(null)
    if (!stripe || !elements) {
      setServerError('Payment is still loading. Try again in a moment.')
      return
    }
    if (!isValid) {
      const all: Partial<Record<keyof FormState, boolean>> = {}
      ;(Object.keys(form) as (keyof FormState)[]).forEach((k) => (all[k] = true))
      setTouched(all)
      return
    }
    if (!cart || cart.items.length === 0) {
      setServerError('Your cart is empty.')
      return
    }

    setSubmitting(true)
    try {
      // 1. Validate Stripe inputs locally before creating the order. This
      //    prevents a stuck pending_payment order on basic card-format errors.
      const submission = await elements.submit()
      if (submission.error) {
        setServerError(mapStripeError(submission.error))
        return
      }

      // 2. Create the order — reserves stock, returns the order_id.
      const body: CheckoutRequest = {
        email: form.email.trim(),
        phone: form.phone.trim() || undefined,
        shipping_method: form.shipping_method,
        address: {
          recipient_name: form.recipient_name.trim(),
          line1: form.line1.trim(),
          line2: form.line2.trim() || undefined,
          city: form.city.trim(),
          region: form.region.trim(),
          postal_code: form.postal_code.trim(),
          country: form.country.trim().toUpperCase(),
        },
      }
      const orderResp = await api.checkout(body)

      // 3. Create the PaymentIntent for the new order.
      const intent = await api.createPaymentIntent(orderResp.order.id)

      // 4. Confirm with Stripe. On success it redirects to return_url.
      const result = await stripe.confirmPayment({
        elements,
        clientSecret: intent.client_secret,
        confirmParams: {
          return_url: `${window.location.origin}/orders/${orderResp.order.id}/confirmation`,
        },
      })

      if (result.error) {
        // Order already exists and the cart is cleared, so retry must
        // happen on /pay against the existing order. Carry the decline
        // reason through router state so the user sees why on arrival.
        navigate(`/orders/${orderResp.order.id}/pay`, {
          replace: true,
          state: { error: mapStripeError(result.error) },
        })
      }
    } catch (e) {
      setServerError(mapPaymentError(e))
    } finally {
      setSubmitting(false)
    }
  }

  if (!cart || cart.items.length === 0) return null

  return (
    <Page width="max-w-6xl">
      <Masthead
        number="03"
        eyebrow="Checkout"
        title="Review and pay."
        caption="A single page. Contact, address, shipping, payment. One submit."
      />

      <div className="grid lg:grid-cols-[1.4fr_1fr] gap-12 lg:gap-20">
        <form onSubmit={handleSubmit} noValidate>
          {isAuthed && prefilledEmail && (
            <p className="mb-10 text-xs text-ink-faint border-l-2 border-rule pl-3 py-1">
              Contact prefilled from your account.
            </p>
          )}

          {serverError && (
            <p
              role="alert"
              className="mb-8 text-sm text-accent border-l-2 border-accent pl-3 py-1"
            >
              {serverError}
            </p>
          )}

          <Section number="01" title="Contact">
            <CField
              label="Email"
              error={err('email')}
              input={
                <input
                  type="email"
                  autoComplete="email"
                  value={form.email}
                  onChange={(e) => update('email', e.target.value)}
                  onBlur={() => markTouched('email')}
                  className={inputCls(err('email'))}
                  style={{ borderRadius: 0 }}
                />
              }
            />
            <CField
              label="Phone (optional)"
              error={err('phone')}
              input={
                <input
                  type="tel"
                  autoComplete="tel"
                  value={form.phone}
                  onChange={(e) => update('phone', e.target.value)}
                  onBlur={() => markTouched('phone')}
                  className={inputCls(err('phone'))}
                  style={{ borderRadius: 0 }}
                />
              }
            />
          </Section>

          <Section number="02" title="Shipping address">
            <CField
              label="Recipient name"
              error={err('recipient_name')}
              input={
                <input
                  autoComplete="name"
                  value={form.recipient_name}
                  onChange={(e) => update('recipient_name', e.target.value)}
                  onBlur={() => markTouched('recipient_name')}
                  className={inputCls(err('recipient_name'))}
                  style={{ borderRadius: 0 }}
                />
              }
            />
            <CField
              label="Address line 1"
              error={err('line1')}
              input={
                <input
                  autoComplete="address-line1"
                  value={form.line1}
                  onChange={(e) => update('line1', e.target.value)}
                  onBlur={() => markTouched('line1')}
                  className={inputCls(err('line1'))}
                  style={{ borderRadius: 0 }}
                />
              }
            />
            <CField
              label="Address line 2 (optional)"
              error={err('line2')}
              input={
                <input
                  autoComplete="address-line2"
                  value={form.line2}
                  onChange={(e) => update('line2', e.target.value)}
                  onBlur={() => markTouched('line2')}
                  className={inputCls(err('line2'))}
                  style={{ borderRadius: 0 }}
                />
              }
            />
            <div className="grid grid-cols-2 gap-6">
              <CField
                label="City"
                error={err('city')}
                input={
                  <input
                    autoComplete="address-level2"
                    value={form.city}
                    onChange={(e) => update('city', e.target.value)}
                    onBlur={() => markTouched('city')}
                    className={inputCls(err('city'))}
                    style={{ borderRadius: 0 }}
                  />
                }
              />
              <CField
                label="State / region"
                error={err('region')}
                input={
                  <input
                    autoComplete="address-level1"
                    value={form.region}
                    onChange={(e) => update('region', e.target.value)}
                    onBlur={() => markTouched('region')}
                    className={inputCls(err('region'))}
                    style={{ borderRadius: 0 }}
                  />
                }
              />
            </div>
            <div className="grid grid-cols-2 gap-6">
              <CField
                label="Postal code"
                error={err('postal_code')}
                input={
                  <input
                    autoComplete="postal-code"
                    value={form.postal_code}
                    onChange={(e) => update('postal_code', e.target.value)}
                    onBlur={() => markTouched('postal_code')}
                    className={inputCls(err('postal_code'))}
                    style={{ borderRadius: 0 }}
                  />
                }
              />
              <CField
                label="Country (ISO-2)"
                hint="Two-letter code, e.g. US, GB, EE."
                error={err('country')}
                input={
                  <input
                    autoComplete="country"
                    maxLength={2}
                    value={form.country}
                    onChange={(e) =>
                      update('country', e.target.value.toUpperCase())
                    }
                    onBlur={() => markTouched('country')}
                    className={inputCls(err('country'))}
                    style={{ borderRadius: 0 }}
                  />
                }
              />
            </div>
          </Section>

          <Section number="03" title="Shipping method">
            <div className="space-y-2">
              {SHIPPING_OPTIONS.map((opt) => {
                const active = form.shipping_method === opt.id
                return (
                  <label
                    key={opt.id}
                    className={`flex items-center justify-between gap-4 px-5 py-4 border cursor-pointer transition-colors ${
                      active
                        ? 'border-ink bg-raised'
                        : 'border-rule hover:border-rule-strong'
                    }`}
                    style={{ borderRadius: 0 }}
                  >
                    <div className="flex items-center gap-4">
                      <input
                        type="radio"
                        name="shipping_method"
                        value={opt.id}
                        checked={active}
                        onChange={() => update('shipping_method', opt.id)}
                        className="accent-current text-ink"
                      />
                      <div>
                        <p className="text-ink">{opt.label}</p>
                        <p className="text-xs text-ink-faint mt-0.5">{opt.eta}</p>
                      </div>
                    </div>
                    <span className="tnum text-ink">{formatPrice(opt.cents)}</span>
                  </label>
                )
              })}
            </div>
          </Section>

          <Section number="04" title="Payment">
            <div className="bg-raised px-5 py-5 border border-rule-strong">
              <PaymentElement options={{ layout: 'tabs' }} />
            </div>
            <p className="text-xs text-ink-faint mt-3 leading-relaxed">
              Test card{' '}
              <span className="tnum text-ink-soft">4242 4242 4242 4242</span>{' '}
              succeeds.{' '}
              <span className="tnum text-ink-soft">4000 0000 0000 9995</span>{' '}
              simulates insufficient funds. Any future date, any CVC.
            </p>
          </Section>

          <button
            type="submit"
            disabled={submitting || !stripe || !elements}
            className="mt-4 bg-accent text-on-accent hover:bg-accent-soft transition-colors px-7 py-3 text-sm tracking-[0.01em] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {submitting ? 'Placing order.' : `Place order · ${formatPrice(total)}`}
          </button>
          <p className="mt-3 text-xs text-ink-faint">
            We'll create your order and charge your card in one step.
          </p>
        </form>

        <aside className="lg:sticky lg:top-8 lg:self-start">
          <p className="uc-tight text-[0.7rem] text-ink-faint mb-4">Summary</p>
          {lineError && (
            <p
              role="alert"
              className="mb-3 text-xs text-accent border-l-2 border-accent pl-2 py-1"
            >
              {lineError}
            </p>
          )}
          <ul className="divide-y divide-rule border-y border-rule">
            {cart.items.map((it) => {
              const busy = lineBusy === it.product_id
              return (
                <li
                  key={it.id}
                  className="grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 py-3 text-sm"
                >
                  <span className="text-ink leading-snug min-w-0">
                    {it.product_name}
                  </span>
                  <span className="tnum text-ink justify-self-end">
                    {formatPrice(it.product_price * it.quantity)}
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => adjust(it.product_id, it.quantity - 1)}
                      disabled={busy || it.quantity <= 1}
                      aria-label={`Decrease ${it.product_name}`}
                      className="w-6 h-6 border border-rule-strong text-ink hover:border-ink hover:text-accent transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
                      style={{ borderRadius: 0 }}
                    >
                      −
                    </button>
                    <span className="w-6 text-center tnum text-ink-soft text-xs">
                      {it.quantity}
                    </span>
                    <button
                      type="button"
                      onClick={() => adjust(it.product_id, it.quantity + 1)}
                      disabled={busy || it.quantity >= it.stock}
                      aria-label={`Increase ${it.product_name}`}
                      className="w-6 h-6 border border-rule-strong text-ink hover:border-ink hover:text-accent transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
                      style={{ borderRadius: 0 }}
                    >
                      +
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => adjust(it.product_id, 0)}
                    disabled={busy}
                    className="text-[0.7rem] text-ink-faint hover:text-accent underline underline-offset-4 justify-self-end disabled:opacity-30 cursor-pointer"
                  >
                    Remove
                  </button>
                </li>
              )
            })}
          </ul>
          <dl className="mt-6 space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-ink-soft">Subtotal</dt>
              <dd className="tnum text-ink">{formatPrice(subtotal)}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-ink-soft">
                Shipping ({shipping.label.toLowerCase()})
              </dt>
              <dd className="tnum text-ink">{formatPrice(shipping.cents)}</dd>
            </div>
            <div className="flex justify-between pt-4 mt-2 border-t border-rule items-baseline">
              <dt className="uc-tight text-[0.7rem] text-ink-faint">Total</dt>
              <dd
                className="font-display tnum text-ink text-[clamp(1.5rem,3vw,2rem)] leading-none font-bold"
              >
                {formatPrice(total)}
              </dd>
            </div>
          </dl>
        </aside>
      </div>
    </Page>
  )
}

function Section({
  number,
  title,
  children,
}: {
  number: string
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="mb-12">
      <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
        <span className="tnum">{number}</span>
        <span aria-hidden className="text-rule-strong mx-2">
          /
        </span>
        {title}
      </h2>
      <div className="space-y-6">{children}</div>
    </section>
  )
}

function CField({
  label,
  hint,
  error,
  input,
}: {
  label: string
  hint?: string
  error?: string
  input: React.ReactNode
}) {
  return (
    <label className="block">
      <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
        {label}
      </span>
      {input}
      {hint && !error && (
        <span className="block mt-1.5 text-xs text-ink-faint">{hint}</span>
      )}
      {error && <span className="block mt-1.5 text-xs text-accent">{error}</span>}
    </label>
  )
}

function inputCls(error?: string) {
  return `w-full bg-raised border-0 border-b px-0 py-2 text-ink placeholder-ink-faint transition-colors focus:border-ink ${
    error ? 'border-accent' : 'border-rule-strong'
  }`
}

