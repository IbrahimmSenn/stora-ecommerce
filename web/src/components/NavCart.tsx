/* NavCart.tsx — cart icon with a count badge.
 *
 * NavLink to /cart. ShoppingBag icon centred, badge as a small pill at the
 * top-right when itemCount > 0. The badge transform-pulses on count change so
 * the persistent signal links visually to the add action. Honors reduced
 * motion.
 */
import { useEffect, useRef, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useCart } from '../cart/useCart'
import { useReducedMotion } from '../lib/motion'
import { ShoppingBag } from './icons'

export function NavCart({ onDark = false }: { onDark?: boolean } = {}) {
  const { itemCount } = useCart()
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
      ? 'Cart, empty'
      : `Cart, ${itemCount} ${itemCount === 1 ? 'item' : 'items'}`

  return (
    <NavLink
      to="/cart"
      aria-label={label}
      className={({ isActive }) =>
        `relative inline-flex h-10 w-10 md:h-9 md:w-9 items-center justify-center transition-colors ${
          onDark
            ? isActive
              ? 'text-on-primary'
              : 'text-on-primary/80 hover:text-on-primary'
            : isActive
              ? 'text-ink'
              : 'text-ink-soft hover:text-ink'
        }`
      }
    >
      <ShoppingBag size={18} strokeWidth={1.5} aria-hidden />
      {itemCount > 0 && (
        <span
          aria-hidden
          className="tnum absolute -top-0.5 -right-0.5 min-w-[1.1rem] h-[1.1rem] px-1 inline-flex items-center justify-center bg-accent text-on-accent text-[0.65rem] leading-none rounded-full"
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
    </NavLink>
  )
}
