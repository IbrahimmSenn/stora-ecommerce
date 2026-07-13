/* Shared Stripe loader + theme palette.
 *
 * Stripe accepts hex/rgb but not OKLCH, so the variables below approximate
 * the design tokens. The Stripe promise is memoised per publishable key so
 * loadStripe fires at most once per page load (per Stripe's own guidance).
 */
import { loadStripe } from '@stripe/stripe-js'
import type { Appearance, Stripe } from '@stripe/stripe-js'

const cache = new Map<string, Promise<Stripe | null>>()

export function getStripe(publishableKey: string) {
  let p = cache.get(publishableKey)
  if (!p) {
    p = loadStripe(publishableKey)
    cache.set(publishableKey, p)
  }
  return p
}

type Theme = 'light' | 'dark'

// Hex approximations of the OKLCH design tokens (web/src/styles/tokens.css):
// cobalt-indigo primary, coral-red danger, and the neutral surface/ink/border
// for each theme. Kept in sync with tokens.css by eye — Stripe Elements only
// accepts hex, so these can't reference the CSS variables directly.
const STRIPE_THEME = {
  light: {
    colorPrimary: '#2e5bda',
    colorBackground: '#fafafa',
    colorText: '#14161b',
    colorDanger: '#cc272e',
    borderColor: '#bbbec3',
  },
  dark: {
    colorPrimary: '#588aff',
    colorBackground: '#0b0d13',
    colorText: '#edeef2',
    colorDanger: '#fc5855',
    borderColor: '#484d58',
  },
} as const

export function stripeAppearance(theme: Theme): Appearance {
  const palette = STRIPE_THEME[theme]
  return {
    theme: 'flat',
    variables: {
      colorPrimary: palette.colorPrimary,
      colorBackground: palette.colorBackground,
      colorText: palette.colorText,
      colorDanger: palette.colorDanger,
      fontFamily:
        '"Hanken Grotesk Variable", "Hanken Grotesk", system-ui, sans-serif',
      borderRadius: '0px',
      fontSizeBase: '15px',
      spacingUnit: '4px',
    },
    rules: {
      '.Input': {
        border: `1px solid ${palette.borderColor}`,
        boxShadow: 'none',
      },
      '.Input:focus': {
        border: `1px solid ${palette.colorText}`,
        boxShadow: 'none',
      },
      '.Label': {
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        fontSize: '11px',
        color: palette.colorText,
        opacity: '0.6',
      },
    },
  }
}
