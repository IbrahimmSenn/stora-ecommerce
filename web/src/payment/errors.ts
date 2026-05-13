/* errors.ts — payment failure → user-facing copy.
 *
 * Maps the named Stripe failure scenarios (insufficient funds, invalid
 * card number, expired card, processing/timeout) to specific, sentence-case
 * messages and falls back to Stripe's own localised text for anything else.
 *
 * Network errors from our own backend (fetch TypeError, 502/504 from the
 * API) get a distinct message so the user knows it's a connection problem
 * and not a card decline.
 */
import type { StripeError } from '@stripe/stripe-js'
import { ApiError } from '../lib/api'

export function mapStripeError(error: StripeError | undefined): string {
  if (!error) return 'Payment failed.'

  const code = error.code ?? ''
  const declineCode = error.decline_code ?? ''

  // Insufficient funds — Stripe sets code='card_declined' with
  // decline_code='insufficient_funds' for the 9995 test card.
  if (declineCode === 'insufficient_funds' || code === 'insufficient_funds') {
    return 'Your card has insufficient funds for this purchase. Use a different card.'
  }

  // Expired card — code='expired_card' (or as a decline_code variant).
  if (code === 'expired_card' || declineCode === 'expired_card') {
    return 'Your card has expired. Use a different card.'
  }

  // Invalid / malformed card number. Stripe Elements emits these
  // synchronously on submit().
  if (
    code === 'invalid_number' ||
    code === 'incomplete_number' ||
    code === 'incorrect_number' ||
    declineCode === 'incorrect_number'
  ) {
    return 'The card number is invalid. Check the digits and try again.'
  }

  // CVC / expiry detail problems — surfaced as separate codes by Elements.
  if (code === 'invalid_cvc' || code === 'incomplete_cvc' || code === 'incorrect_cvc') {
    return 'The security code (CVC) is invalid.'
  }
  if (code === 'invalid_expiry_month' || code === 'invalid_expiry_year' || code === 'incomplete_expiry') {
    return 'The expiry date is invalid.'
  }

  // Processing / gateway timeout. Stripe surfaces network issues between
  // them and the issuer as `processing_error`.
  if (code === 'processing_error' || declineCode === 'processing_error') {
    return 'The card issuer could not process the payment. Try again in a moment.'
  }

  // Generic decline. Use Stripe's own message — they localise it and
  // tailor it per decline reason.
  if (code === 'card_declined') {
    return error.message ?? 'Your card was declined.'
  }

  return error.message ?? 'Payment failed.'
}

/** True when an exception is a fetch network failure (server unreachable,
 *  DNS failure, request aborted). Distinct from a 4xx/5xx API response. */
export function isNetworkError(e: unknown): boolean {
  if (e instanceof TypeError) {
    return /fetch|network|failed/i.test(e.message)
  }
  if (e instanceof DOMException && e.name === 'AbortError') {
    return true
  }
  return false
}

/** Convert any thrown value from a payment-related call into user-facing
 *  copy. Stripe errors get the specific mapping above; ApiError uses the
 *  server's message; network errors get a connection-specific line. */
export function mapPaymentError(e: unknown): string {
  if (isNetworkError(e)) {
    return 'Network error. Check your connection and try again.'
  }
  if (e instanceof ApiError) {
    if (e.status >= 502 && e.status <= 504) {
      return 'The payment gateway timed out. Try again in a moment.'
    }
    return e.message
  }
  if (e && typeof e === 'object' && 'type' in e && 'code' in e) {
    return mapStripeError(e as StripeError)
  }
  return 'Something went wrong. Please try again.'
}
