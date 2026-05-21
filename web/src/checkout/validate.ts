// Mirrors the server-side validator/v10 rules in
// internal/orders/model.go — keep these in sync.
export type CheckoutFormState = {
  email: string
  phone: string
  shipping_method: 'standard' | 'express'
  recipient_name: string
  line1: string
  line2: string
  city: string
  region: string
  postal_code: string
  country: string
}

export function validateCheckout(
  f: CheckoutFormState,
): Partial<Record<keyof CheckoutFormState, string>> {
  const e: Partial<Record<keyof CheckoutFormState, string>> = {}
  const email = f.email.trim()
  if (!email) e.email = 'Required.'
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) e.email = 'Invalid email.'

  if (f.phone.trim()) {
    const p = f.phone.trim()
    if (p.length < 7 || p.length > 20) e.phone = '7–20 characters.'
  }

  if (!f.recipient_name.trim()) e.recipient_name = 'Required.'
  if (!f.line1.trim()) e.line1 = 'Required.'
  if (!f.city.trim()) e.city = 'Required.'
  if (!f.region.trim()) e.region = 'Required.'

  const postal = f.postal_code.trim()
  if (!postal) e.postal_code = 'Required.'
  else if (postal.length < 3 || postal.length > 12)
    e.postal_code = '3–12 characters.'

  const country = f.country.trim()
  if (!country) e.country = 'Required.'
  else if (!/^[A-Za-z]{2}$/.test(country)) e.country = 'Two-letter code.'

  return e
}
