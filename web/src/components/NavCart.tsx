/* NavCart.tsx — cart icon with a count badge.
 *
 * NavLink to /cart. ShoppingBag icon centred, badge as a small pill at the
 * top-right when itemCount > 0. The badge transform-pulses on count change so
 * the persistent signal links visually to the add action. Honors reduced
 * motion.
 */
import { useEffect, useRef, useState } from 'react'
import { useCart } from '../cart/useCart'
import { useCartPanel } from '../cart/useCartPanel'
import { useReducedMotion } from '../lib/motion'
import { ShoppingCart } from './icons'

export function NavCart({ onDark = false }: { onDark?: boolean } = {}) {
  const { itemCount } = useCart()
  const { open } = useCartPanel()
  const reduced = useReducedMotion()
  const [pulse, setPulse] = useState(false)
  const prev = useRef(itemCount)

  useEffect(() => {
    if (reduced) {
      prev.current = itemCount
      return
    }
    if (itemCount !== prev.current) {
      prev.current = itemCount
      setPulse(true)
      const t = window.setTimeout(() => setPulse(false), 240)
      return () => window.clearTimeout(t)
    }
  }, [itemCount, reduced])

  const label =
    itemCount === 0
      ? 'Open cart, empty'
      : `Open cart, ${itemCount} ${itemCount === 1 ? 'item' : 'items'}`

  return (
    <button
      type="button"
      onClick={(e) => open(e.currentTarget)}
      aria-label={label}
      aria-haspopup="dialog"
      className={`relative inline-flex h-12 w-12 md:h-11 md:w-11 items-center justify-center transition-colors cursor-pointer ${
        onDark
          ? 'text-on-primary/80 hover:text-on-primary'
          : 'text-ink-soft hover:text-ink'
      }`}
    >
      <ShoppingCart size={26} strokeWidth={1.75} aria-hidden />
      {itemCount > 0 && (
        <span
          aria-hidden
          className="tnum absolute top-0.5 left-1 min-w-[1.25rem] h-[1.25rem] px-1 inline-flex items-center justify-center bg-accent text-on-accent text-[0.7rem] font-semibold leading-none rounded-full"
          style={{
            transform: pulse ? 'scale(1.15)' : 'scale(1)',
            transition: reduced
              ? 'none'
              : 'transform 240ms var(--ease-out-quart)',
            transformOrigin: 'center',
          }}
        >
          {itemCount > 99 ? '99+' : itemCount}
        </span>
      )}
    </button>
  )
}
