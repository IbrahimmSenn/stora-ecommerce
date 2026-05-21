import { describe, expect, it } from 'vitest'
import { validateCheckout } from './validate'
import type { CheckoutFormState } from './validate'

function valid(overrides: Partial<CheckoutFormState> = {}): CheckoutFormState {
  return {
    email: 'shopper@example.com',
    phone: '5551234567',
    shipping_method: 'standard',
    recipient_name: 'Buyer',
    line1: '1 Demo St',
    line2: '',
    city: 'Townsville',
    region: 'TS',
    postal_code: '00000',
    country: 'US',
    ...overrides,
  }
}

describe('validateCheckout', () => {
  it('accepts a fully valid form', () => {
    expect(validateCheckout(valid())).toEqual({})
  })

  it('rejects missing required fields', () => {
    const errors = validateCheckout(
      valid({
        email: '',
        recipient_name: '',
        line1: '',
        city: '',
        region: '',
        postal_code: '',
        country: '',
      }),
    )
    expect(errors.email).toBe('Required.')
    expect(errors.recipient_name).toBe('Required.')
    expect(errors.line1).toBe('Required.')
    expect(errors.city).toBe('Required.')
    expect(errors.region).toBe('Required.')
    expect(errors.postal_code).toBe('Required.')
    expect(errors.country).toBe('Required.')
  })

  it('rejects malformed email', () => {
    expect(validateCheckout(valid({ email: 'not-an-email' })).email).toBe(
      'Invalid email.',
    )
    expect(validateCheckout(valid({ email: 'a@b' })).email).toBe(
      'Invalid email.',
    )
    expect(validateCheckout(valid({ email: '@example.com' })).email).toBe(
      'Invalid email.',
    )
  })

  it('treats phone as optional but enforces length when present', () => {
    expect(validateCheckout(valid({ phone: '' })).phone).toBeUndefined()
    expect(validateCheckout(valid({ phone: '12345' })).phone).toBe(
      '7–20 characters.',
    )
    expect(
      validateCheckout(valid({ phone: '1'.repeat(21) })).phone,
    ).toBe('7–20 characters.')
  })

  it('enforces postal code length range', () => {
    expect(validateCheckout(valid({ postal_code: 'AB' })).postal_code).toBe(
      '3–12 characters.',
    )
    expect(
      validateCheckout(valid({ postal_code: 'A'.repeat(13) })).postal_code,
    ).toBe('3–12 characters.')
    expect(
      validateCheckout(valid({ postal_code: '12345' })).postal_code,
    ).toBeUndefined()
  })

  it('enforces two-letter alpha country code', () => {
    expect(validateCheckout(valid({ country: 'USA' })).country).toBe(
      'Two-letter code.',
    )
    expect(validateCheckout(valid({ country: '12' })).country).toBe(
      'Two-letter code.',
    )
    expect(validateCheckout(valid({ country: 'GB' })).country).toBeUndefined()
  })

  it('trims whitespace before validating', () => {
    expect(
      validateCheckout(valid({ email: '  shopper@example.com  ' })).email,
    ).toBeUndefined()
    expect(
      validateCheckout(valid({ recipient_name: '   ' })).recipient_name,
    ).toBe('Required.')
  })
})
