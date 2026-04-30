import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useCart } from '../cart/useCart'
import { useAuth } from '../auth/useAuth'
import { api, ApiError, formatPrice } from '../lib/api'
import type { CheckoutRequest, ShippingMethod } from '../lib/api'

type FormState = {
  email: string
  phone: string
  shipping_method: ShippingMethod
  recipient_name: string
  line1: string
  line2: string
  city: string
  region: string
  postal_code: string
  country: string
}

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
  const { cart, loading, refresh } = useCart()
  const { email: authedEmail } = useAuth()
  const navigate = useNavigate()

  const [form, setForm] = useState<FormState>(initial)
  const [touched, setTouched] = useState<Partial<Record<keyof FormState, boolean>>>({})
  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  useEffect(() => {
    if (authedEmail && !form.email) {
      setForm((f) => ({ ...f, email: authedEmail }))
    }
  }, [authedEmail, form.email])

  const shipping = useMemo(
    () => SHIPPING_OPTIONS.find((o) => o.id === form.shipping_method)!,
    [form.shipping_method],
  )
  const subtotal = cart?.total ?? 0
  const total = subtotal + shipping.cents

  const errors = useMemo(() => validate(form), [form])
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
    if (!isValid) {
      // touch every field so errors become visible
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
      const resp = await api.checkout(body)
      await refresh()
      navigate(`/orders/${resp.order.id}/confirmation`, { replace: true })
    } catch (e) {
      if (e instanceof ApiError) {
        setServerError(e.message)
      } else {
        setServerError('Something went wrong. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <p className="p-8">Loading…</p>

  if (!cart || cart.items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto p-8 text-center">
        <h1 className="text-2xl font-semibold mb-2">Nothing to check out</h1>
        <p className="text-gray-600 mb-4">Your cart is empty.</p>
        <Link to="/" className="underline">
          Browse products
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-6xl mx-auto px-6 py-12">
      <header className="mb-12">
        <p className="text-xs uppercase tracking-widest text-gray-500">Checkout</p>
        <h1 className="text-4xl font-semibold mt-2">Review and confirm</h1>
      </header>

      <div className="grid lg:grid-cols-[1.4fr_1fr] gap-16">
        <form onSubmit={handleSubmit} noValidate>
          {serverError && (
            <div
              role="alert"
              className="mb-8 px-4 py-3 border border-red-200 bg-red-50 text-red-800 text-sm rounded"
            >
              {serverError}
            </div>
          )}

          <Section number="01" title="Contact">
            <Field
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
                />
              }
            />
            <Field
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
                />
              }
            />
          </Section>

          <Section number="02" title="Shipping address">
            <Field
              label="Recipient name"
              error={err('recipient_name')}
              input={
                <input
                  autoComplete="name"
                  value={form.recipient_name}
                  onChange={(e) => update('recipient_name', e.target.value)}
                  onBlur={() => markTouched('recipient_name')}
                  className={inputCls(err('recipient_name'))}
                />
              }
            />
            <Field
              label="Address line 1"
              error={err('line1')}
              input={
                <input
                  autoComplete="address-line1"
                  value={form.line1}
                  onChange={(e) => update('line1', e.target.value)}
                  onBlur={() => markTouched('line1')}
                  className={inputCls(err('line1'))}
                />
              }
            />
            <Field
              label="Address line 2 (optional)"
              error={err('line2')}
              input={
                <input
                  autoComplete="address-line2"
                  value={form.line2}
                  onChange={(e) => update('line2', e.target.value)}
                  onBlur={() => markTouched('line2')}
                  className={inputCls(err('line2'))}
                />
              }
            />
            <div className="grid grid-cols-2 gap-4">
              <Field
                label="City"
                error={err('city')}
                input={
                  <input
                    autoComplete="address-level2"
                    value={form.city}
                    onChange={(e) => update('city', e.target.value)}
                    onBlur={() => markTouched('city')}
                    className={inputCls(err('city'))}
                  />
                }
              />
              <Field
                label="State / region"
                error={err('region')}
                input={
                  <input
                    autoComplete="address-level1"
                    value={form.region}
                    onChange={(e) => update('region', e.target.value)}
                    onBlur={() => markTouched('region')}
                    className={inputCls(err('region'))}
                  />
                }
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field
                label="Postal code"
                error={err('postal_code')}
                input={
                  <input
                    autoComplete="postal-code"
                    value={form.postal_code}
                    onChange={(e) => update('postal_code', e.target.value)}
                    onBlur={() => markTouched('postal_code')}
                    className={inputCls(err('postal_code'))}
                  />
                }
              />
              <Field
                label="Country (ISO-2)"
                hint="Two-letter code, e.g. US, GB, EE"
                error={err('country')}
                input={
                  <input
                    autoComplete="country"
                    maxLength={2}
                    value={form.country}
                    onChange={(e) => update('country', e.target.value.toUpperCase())}
                    onBlur={() => markTouched('country')}
                    className={inputCls(err('country'))}
                  />
                }
              />
            </div>
          </Section>

          <Section number="03" title="Shipping method">
            <div className="space-y-2">
              {SHIPPING_OPTIONS.map((opt) => (
                <label
                  key={opt.id}
                  className={`flex items-center justify-between gap-4 px-4 py-3 border cursor-pointer ${
                    form.shipping_method === opt.id
                      ? 'border-gray-900'
                      : 'border-gray-200 hover:border-gray-400'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <input
                      type="radio"
                      name="shipping_method"
                      value={opt.id}
                      checked={form.shipping_method === opt.id}
                      onChange={() => update('shipping_method', opt.id)}
                      className="accent-gray-900"
                    />
                    <div>
                      <p className="font-medium">{opt.label}</p>
                      <p className="text-sm text-gray-500">{opt.eta}</p>
                    </div>
                  </div>
                  <span className="tabular-nums">{formatPrice(opt.cents)}</span>
                </label>
              ))}
            </div>
          </Section>

          <button
            type="submit"
            disabled={submitting}
            className="mt-10 w-full sm:w-auto px-8 py-3 bg-gray-900 text-white text-sm uppercase tracking-wider disabled:opacity-50"
          >
            {submitting ? 'Placing order…' : 'Place order'}
          </button>
          <p className="mt-3 text-xs text-gray-500">
            Payment is collected on the next milestone — your order will be created
            with status <span className="font-medium">pending payment</span>.
          </p>
        </form>

        <aside className="lg:sticky lg:top-8 lg:self-start">
          <p className="text-xs uppercase tracking-widest text-gray-500 mb-4">Summary</p>
          <ul className="divide-y border-t border-b">
            {cart.items.map((it) => (
              <li key={it.id} className="flex justify-between gap-4 py-3 text-sm">
                <span className="flex-1">
                  {it.product_name}
                  <span className="text-gray-500"> × {it.quantity}</span>
                </span>
                <span className="tabular-nums">
                  {formatPrice(it.product_price * it.quantity)}
                </span>
              </li>
            ))}
          </ul>
          <dl className="mt-6 space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Subtotal</dt>
              <dd className="tabular-nums">{formatPrice(subtotal)}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Shipping ({shipping.label.toLowerCase()})</dt>
              <dd className="tabular-nums">{formatPrice(shipping.cents)}</dd>
            </div>
            <div className="flex justify-between pt-3 border-t text-base font-semibold">
              <dt>Total</dt>
              <dd className="tabular-nums">{formatPrice(total)}</dd>
            </div>
          </dl>
        </aside>
      </div>
    </div>
  )
}

// --- helpers ---

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
    <section className="mb-10">
      <div className="flex items-baseline gap-3 mb-4">
        <span className="text-xs tabular-nums text-gray-400">{number}</span>
        <h2 className="text-lg font-medium">{title}</h2>
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  )
}

function Field({
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
    <label className="block text-sm">
      <span className="block text-gray-600 mb-1">{label}</span>
      {input}
      {hint && !error && <span className="block mt-1 text-xs text-gray-400">{hint}</span>}
      {error && <span className="block mt-1 text-xs text-red-600">{error}</span>}
    </label>
  )
}

function inputCls(error?: string) {
  return `w-full border px-3 py-2 outline-none focus:border-gray-900 ${
    error ? 'border-red-400' : 'border-gray-300'
  }`
}

// Matches the server-side validator/v10 rules in
// internal/orders/model.go — keep these in sync.
function validate(f: FormState): Partial<Record<keyof FormState, string>> {
  const e: Partial<Record<keyof FormState, string>> = {}
  const email = f.email.trim()
  if (!email) e.email = 'required'
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) e.email = 'invalid email'

  if (f.phone.trim()) {
    const p = f.phone.trim()
    if (p.length < 7 || p.length > 20) e.phone = '7–20 characters'
  }

  if (!f.recipient_name.trim()) e.recipient_name = 'required'
  if (!f.line1.trim()) e.line1 = 'required'
  if (!f.city.trim()) e.city = 'required'
  if (!f.region.trim()) e.region = 'required'

  const postal = f.postal_code.trim()
  if (!postal) e.postal_code = 'required'
  else if (postal.length < 3 || postal.length > 12) e.postal_code = '3–12 characters'

  const country = f.country.trim()
  if (!country) e.country = 'required'
  else if (!/^[A-Za-z]{2}$/.test(country)) e.country = 'two-letter code'

  return e
}
