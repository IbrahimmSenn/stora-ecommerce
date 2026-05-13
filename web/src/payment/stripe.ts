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

const STRIPE_THEME = {
  light: {
    colorPrimary: '#c95f3f',
    colorBackground: '#fafaf9',
    colorText: '#241f1d',
    colorDanger: '#c95f3f',
    borderColor: '#bdb5b1',
  },
  dark: {
    colorPrimary: '#e3805f',
    colorBackground: '#1a1d22',
    colorText: '#ece9e3',
    colorDanger: '#e3805f',
    borderColor: '#4f5460',
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
