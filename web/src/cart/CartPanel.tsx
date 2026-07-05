/* CartPanel.tsx — the signature cart transition.
 *
 * Right-side surface panel that slides in when an item is added to cart.
 * Backdrop is a faint ink wash, not a heavy dim. The just-added line gets a
 * recede wash (an inert accent overlay that fades to 0 over 1.4s) — never a
 * coloured stripe. Transform + opacity only. Honors prefers-reduced-motion.
 */
import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { Link, useLocation } from 'react-router-dom'
import { useCart } from './useCart'
import { useCartPanel } from './useCartPanel'
import { formatPrice } from '../lib/api'
import type { CartItem } from '../lib/api'
import { useReducedMotion } from '../lib/motion'

const FOCUSABLE =
  'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"]), input:not([disabled]), select:not([disabled])'

export function CartPanel() {
  const { isOpen, added, close } = useCartPanel()
  const { cart } = useCart()
  const reduced = useReducedMotion()
  const panelRef = useRef<HTMLDivElement | null>(null)
  const location = useLocation()

  // Close on route change. The user navigated — the panel is stale context.
  useEffect(() => {
    if (isOpen) close()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  // ESC key closes. Lock body scroll while open.
  useEffect(() => {
    if (!isOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', onKey)
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = previousOverflow
    }
  }, [isOpen, close])

  // Move focus into the panel on open.
  useEffect(() => {
    if (!isOpen) return
    const t = window.setTimeout(() => {
      const node = panelRef.current
      if (!node) return
      const focusables = node.querySelectorAll<HTMLElement>(FOCUSABLE)
      if (focusables[0]) focusables[0].focus()
    }, 50)
    return () => window.clearTimeout(t)
  }, [isOpen])

  // Focus trap.
  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key !== 'Tab') return
    const node = panelRef.current
    if (!node) return
    const focusables = Array.from(node.querySelectorAll<HTMLElement>(FOCUSABLE))
    if (focusables.length === 0) return
    const first = focusables[0]
    const last = focusables[focusables.length - 1]
    const active = document.activeElement as HTMLElement | null
    if (e.shiftKey && active === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && active === last) {
      e.preventDefault()
      first.focus()
    }
  }

  const itemCount = cart?.items.reduce((sum, it) => sum + it.quantity, 0) ?? 0

  // Render even when closed so transitions play; toggle visibility via state.
  return createPortal(
    <div
      aria-hidden={!isOpen}
      className={`fixed inset-0 z-50 ${isOpen ? 'pointer-events-auto' : 'pointer-events-none'}`}
    >
      <button
        type="button"
        aria-label="Close cart panel"
        tabIndex={-1}
        onClick={close}
        className="absolute inset-0 cursor-default"
        style={{
          background: 'oklch(0.18 0.01 25 / 0.16)',
          opacity: isOpen ? 1 : 0,
          transition: reduced
            ? 'none'
            : `opacity ${isOpen ? 240 : 200}ms var(--ease-out-quart)`,
        }}
      />

      <aside
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Added to cart"
        onKeyDown={handleKeyDown}
        className="absolute right-0 top-0 h-full bg-surface border-l border-rule flex flex-col"
        style={{
          width: 'clamp(20rem, 30vw, 28rem)',
          maxWidth: '100vw',
          transform: isOpen ? 'translateX(0)' : 'translateX(100%)',
          transition: reduced
            ? 'none'
            : `transform ${isOpen ? 360 : 280}ms var(--ease-out-quart)`,
          willChange: reduced ? undefined : 'transform',
        }}
      >
        <PanelContent
          isOpen={isOpen}
          reduced={reduced}
          added={added}
          items={cart?.items ?? []}
          subtotal={cart?.total ?? 0}
          itemCount={itemCount}
          onClose={close}
        />
      </aside>
    </div>,
    document.body,
  )
}

function PanelContent({
  isOpen,
  reduced,
  added,
  items,
  subtotal,
  itemCount,
  onClose,
}: {
  isOpen: boolean
  reduced: boolean
  added: ReturnType<typeof useCartPanel>['added']
  items: CartItem[]
  subtotal: number
  itemCount: number
  onClose: () => void
}) {
  // Stagger children when the panel opens. 80ms tick, starting at 120ms so the
  // first child finishes after the panel slide settles.
  const enter = (i: number) => ({
    opacity: isOpen ? 1 : 0,
    transform: isOpen ? 'translateY(0)' : 'translateY(8px)',
    transition: reduced
      ? 'none'
      : `opacity 480ms var(--ease-out-quart) ${120 + i * 80}ms, transform 480ms var(--ease-out-quart) ${120 + i * 80}ms`,
  })

  const isEmpty = items.length === 0

  return (
    <>
      <header className="px-8 pt-10 pb-6 flex items-baseline justify-between gap-4">
        <div style={enter(0)}>
          <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">Cart</p>
          <h2 className="font-display text-xl text-ink leading-none font-bold">
            {added ? 'Added to cart' : 'Your cart'}
          </h2>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="text-sm text-ink-soft hover:text-ink underline underline-offset-4 cursor-pointer"
          style={enter(0)}
        >
          Close
        </button>
      </header>

      {isEmpty ? (
        <div className="px-8 pb-10 flex-1" style={enter(1)} aria-live="polite">
          <p className="text-sm text-ink-soft border-t border-rule pt-6">
            Your cart is empty.
          </p>
        </div>
      ) : (
        <ul
          className="px-8 flex-1 overflow-y-auto divide-y divide-rule border-y border-rule"
          style={enter(1)}
          aria-live="polite"
        >
          {items.map((it) => {
            const justAdded = added?.productId === it.product_id
            return (
              <li key={it.id} className="relative py-4 flex items-center gap-4">
                {justAdded && <Wash isOpen={isOpen} reduced={reduced} />}
                <div className="h-14 w-14 bg-sunken shrink-0 overflow-hidden" aria-hidden>
                  {it.image_url ? (
                    <img
                      src={it.image_url}
                      alt=""
                      loading="lazy"
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center px-1">
                      <span className="text-[0.5rem] text-ink-faint uc-tight text-center leading-tight line-clamp-2">
                        {it.product_name}
                      </span>
                    </div>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <Link
                    to={`/product/${it.product_id}`}
                    className="text-ink truncate block hover:text-accent transition-colors"
                  >
                    {it.product_name}
                  </Link>
                  <p className="text-xs text-ink-faint tnum mt-1">
                    {formatPrice(it.product_price)} × {it.quantity}
                  </p>
                </div>
                <p className="tnum text-ink shrink-0">
                  {formatPrice(it.product_price * it.quantity)}
                </p>
              </li>
            )
          })}
        </ul>
      )}

      <div className="px-8 mt-auto pt-6 pb-10 flex flex-col gap-6" style={enter(2)}>
        {!isEmpty && (
          <dl className="flex items-baseline justify-between">
            <dt className="uc-tight text-[0.7rem] text-ink-faint">
              Subtotal{' '}
              <span className="text-rule-strong mx-1" aria-hidden>
                /
              </span>{' '}
              <span className="tnum">{itemCount}</span>{' '}
              {itemCount === 1 ? 'item' : 'items'}
            </dt>
            <dd className="font-display tnum text-ink text-[clamp(1.25rem,2.5vw,1.75rem)] leading-none font-bold">
              {formatPrice(subtotal)}
            </dd>
          </dl>
        )}

        <div className="flex flex-col gap-2">
          {!isEmpty && (
            <Link
              to="/cart"
              className="bg-accent text-on-accent hover:bg-accent-soft transition-colors px-5 py-3 text-sm tracking-[0.01em] text-center"
            >
              View cart
            </Link>
          )}
          <button
            type="button"
            onClick={onClose}
            className="text-sm text-ink-soft hover:text-ink underline underline-offset-4 text-center cursor-pointer"
          >
            Continue shopping
          </button>
        </div>
      </div>
    </>
  )
}

/** A transparent accent overlay that fades from 6% to 0 once the panel
 *  arrives. Pure opacity — no border, no transform of the underlying line. */
function Wash({ isOpen, reduced }: { isOpen: boolean; reduced: boolean }) {
  const ref = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!isOpen || reduced) return
    const node = ref.current
    if (!node) return
    // Start at 0.06 and decay to 0 over 1.4s. Begin 360ms after open so the
    // panel has visibly settled before the wash recedes.
    node.style.opacity = '0.06'
    const t = window.setTimeout(() => {
      if (node) node.style.opacity = '0'
    }, 360)
    return () => window.clearTimeout(t)
  }, [isOpen, reduced])

  return (
    <div
      ref={ref}
      aria-hidden
      className="absolute inset-0 pointer-events-none bg-accent"
      style={{
        opacity: 0,
        transition: reduced
          ? 'none'
          : 'opacity 1400ms var(--ease-out-quart)',
      }}
    />
  )
}
