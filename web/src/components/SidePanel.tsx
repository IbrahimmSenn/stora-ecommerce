/* SidePanel.tsx — slide-in drawer from the right.
 *
 * Triggered by the hamburger in Nav. Backdrop is a faint ink wash; the panel
 * is full-height surface with a 1px rule on its left edge. Transform-only
 * transition; reduced-motion swaps for instant state changes. Focus trap +
 * body scroll lock while open, mirrors CartPanel conventions.
 *
 * Contents (top → bottom):
 *   01 / Categories — links to /shop/<slug>
 *   02 / Navigate   — Shop, Orders
 *   03 / Theme      — Sun/Moon icon toggle
 *   Auth block      — Register/Log in OR email/Account/Log out
 */
import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useCart } from '../cart/useCart'
import { api } from '../lib/api'
import type { Category } from '../lib/api'
import { useReducedMotion } from '../lib/motion'
import { useSidePanel } from './useSidePanel'
import { NavSearch } from './NavSearch'
import { ThemeToggle } from './ThemeToggle'
import { X } from './icons'

const FOCUSABLE =
  'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"]), input:not([disabled]), select:not([disabled])'

export function SidePanel() {
  const { isOpen, close } = useSidePanel()
  const { isAuthed, email, logout } = useAuth()
  const { refresh: refreshCart } = useCart()
  const navigate = useNavigate()
  const reduced = useReducedMotion()
  const panelRef = useRef<HTMLDivElement | null>(null)
  const closeButtonRef = useRef<HTMLButtonElement | null>(null)
  const location = useLocation()
  const [categories, setCategories] = useState<Category[]>([])

  // Load categories once on mount. Static reference data; small payload.
  useEffect(() => {
    let cancelled = false
    api
      .listCategories()
      .then((cats) => {
        if (!cancelled) setCategories(cats)
      })
      .catch(() => {
        // Categories failing is non-fatal — the rest of the panel still works.
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Close on route change.
  useEffect(() => {
    if (isOpen) close()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  // ESC closes. Body scroll lock while open.
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

  // Focus the close button on open. Avoid landing on the search input —
  // that auto-pops the on-screen keyboard on touch, which is intrusive when
  // the user opened the panel to browse categories rather than to search.
  useEffect(() => {
    if (!isOpen) return
    const t = window.setTimeout(() => {
      closeButtonRef.current?.focus()
    }, 50)
    return () => window.clearTimeout(t)
  }, [isOpen])

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

  async function handleLogout() {
    logout()
    close()
    await refreshCart()
    navigate('/')
  }

  return createPortal(
    <div
      aria-hidden={!isOpen}
      className={`fixed inset-0 z-50 ${isOpen ? 'pointer-events-auto' : 'pointer-events-none'}`}
    >
      <button
        type="button"
        aria-label="Close menu"
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
        aria-label="Menu"
        onKeyDown={handleKeyDown}
        className="absolute right-0 top-0 h-full bg-surface border-l border-rule flex flex-col"
        style={{
          width: 'clamp(20rem, 30vw, 22.5rem)',
          maxWidth: '100vw',
          transform: isOpen ? 'translateX(0)' : 'translateX(100%)',
          transition: reduced
            ? 'none'
            : `transform ${isOpen ? 360 : 280}ms var(--ease-out-quart)`,
          willChange: reduced ? undefined : 'transform',
        }}
      >
        <header className="px-8 pt-10 pb-6 flex items-baseline justify-between gap-4">
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">Menu</p>
            <h2 className="font-display text-xl text-ink leading-none font-bold">
              Browse.
            </h2>
          </div>
          <button
            ref={closeButtonRef}
            type="button"
            onClick={close}
            aria-label="Close menu"
            className="inline-flex h-9 w-9 items-center justify-center text-ink-soft hover:text-ink transition-colors cursor-pointer"
          >
            <X size={18} strokeWidth={1.5} aria-hidden />
          </button>
        </header>

        <div className="flex-1 overflow-y-auto px-8 py-6 flex flex-col gap-10">
          <Section number="00" label="Search">
            <NavSearch fullWidth onCommit={close} />
          </Section>

          <Section number="01" label="Categories">
            {categories.length === 0 ? (
              <p className="text-sm text-ink-faint">Loading.</p>
            ) : (
              <ul className="flex flex-col">
                {categories.map((c) => {
                  const to = `/shop/${c.slug}`
                  return (
                    <li key={c.id}>
                      <PanelLink to={to} active={location.pathname === to}>
                        {c.name}
                      </PanelLink>
                    </li>
                  )
                })}
              </ul>
            )}
          </Section>

          <Section number="02" label="Navigate">
            <ul className="flex flex-col">
              <li>
                <PanelLink to="/">Shop</PanelLink>
              </li>
              <li>
                <PanelLink to="/orders">Orders</PanelLink>
              </li>
            </ul>
          </Section>

          <Section number="03" label="Theme">
            <ThemeToggle />
          </Section>
        </div>

        <footer className="border-t border-rule px-8 py-6 flex flex-col gap-2">
          {isAuthed ? (
            <>
              {email && (
                <p className="text-[0.7rem] text-ink-faint truncate mb-2">
                  {email}
                </p>
              )}
              <PanelLink to="/account">Account</PanelLink>
              <PanelButton onClick={handleLogout}>Log out</PanelButton>
            </>
          ) : (
            <>
              <PanelLink to="/register">Register</PanelLink>
              <PanelLink to="/login">Log in</PanelLink>
            </>
          )}
        </footer>
      </aside>
    </div>,
    document.body,
  )
}

function Section({
  number,
  label,
  children,
}: {
  number: string
  label: string
  children: React.ReactNode
}) {
  return (
    <section>
      <p className="uc-tight text-[0.7rem] text-ink-faint mb-4">
        <span className="tnum">{number}</span>
        <span aria-hidden className="text-rule-strong mx-2">
          /
        </span>
        {label}
      </p>
      {children}
    </section>
  )
}

function PanelLink({
  to,
  active = false,
  children,
}: {
  to: string
  active?: boolean
  children: React.ReactNode
}) {
  return (
    <Link
      to={to}
      aria-current={active ? 'page' : undefined}
      className={`block py-2 text-sm transition-colors ${
        active ? 'text-ink' : 'text-ink-soft hover:text-ink'
      }`}
    >
      {children}
    </Link>
  )
}

function PanelButton({
  onClick,
  children,
}: {
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="block w-full text-left py-2 text-sm text-ink-soft hover:text-ink transition-colors cursor-pointer"
    >
      {children}
    </button>
  )
}
