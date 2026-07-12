/* NavProfile.tsx — account entry in the nav.
 *
 * Logged out: person icon toggles a Register / Log in dropdown.
 * Logged in: name (or email prefix) + icon link straight to /account, with a
 * caret beside them toggling a dropdown (name/email header, Account, Log out).
 *
 * Dropdown closes on outside click, ESC, or route change. The trigger refocus
 * on close keeps keyboard flow stable.
 */
import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useCart } from '../cart/useCart'
import { User, ChevronDown } from './icons'

export function NavProfile({ onDark = false }: { onDark?: boolean } = {}) {
  const { isAuthed, email, name, logout } = useAuth()
  const { refresh } = useCart()
  const navigate = useNavigate()
  const location = useLocation()
  const wrapperRef = useRef<HTMLDivElement | null>(null)
  const buttonRef = useRef<HTMLButtonElement | null>(null)
  const [open, setOpen] = useState(false)

  // Close on route change. Setting state from a path-change effect is the
  // simplest expression of "menu is stale, hide it" — the alternative would
  // be wiring a path-change listener to every <Link> click, which is worse.
  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setOpen(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  // Close on ESC + outside click.
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setOpen(false)
        buttonRef.current?.focus()
      }
    }
    function onPointer(e: MouseEvent) {
      if (!wrapperRef.current) return
      if (!wrapperRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    document.addEventListener('mousedown', onPointer)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.removeEventListener('mousedown', onPointer)
    }
  }, [open])

  async function handleLogout() {
    logout()
    setOpen(false)
    await refresh()
    navigate('/')
  }

  const toneCls = (active: boolean) =>
    onDark
      ? active
        ? 'text-on-primary'
        : 'text-on-primary/80 hover:text-on-primary'
      : active
        ? 'text-ink'
        : 'text-ink-soft hover:text-ink'

  const label = name ?? email?.split('@')[0] ?? ''

  return (
    <div ref={wrapperRef} className="relative flex items-center">
      {isAuthed ? (
        <>
          <Link
            to="/account"
            aria-label={`Account, ${label}`}
            className={`inline-flex h-10 md:h-9 items-center justify-center gap-2 w-10 md:w-auto md:px-2 transition-colors ${toneCls(false)}`}
          >
            <span className="hidden md:inline text-xs max-w-[14rem] truncate">
              {label}
            </span>
            <User size={18} strokeWidth={1.5} aria-hidden />
          </Link>
          <button
            ref={buttonRef}
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-haspopup="menu"
            aria-expanded={open}
            aria-label="Account menu"
            className={`inline-flex h-10 md:h-9 w-6 items-center justify-center transition-colors ${toneCls(open)}`}
          >
            <ChevronDown
              size={14}
              strokeWidth={1.5}
              aria-hidden
              style={{
                transform: open ? 'rotate(180deg)' : 'none',
                transition: 'transform 150ms var(--ease-out-quart)',
              }}
            />
          </button>
        </>
      ) : (
        <button
          ref={buttonRef}
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-label="Account"
          className={`inline-flex h-10 md:h-9 w-10 md:w-9 items-center justify-center transition-colors ${toneCls(open)}`}
        >
          <User size={18} strokeWidth={1.5} aria-hidden />
        </button>
      )}

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full mt-2 z-40 min-w-[14rem] bg-surface border border-rule shadow-[0_8px_24px_oklch(0.18_0.01_25/0.08)]"
        >
          {isAuthed ? (
            <>
              <div className="px-4 pt-3 pb-2">
                {name && <p className="text-sm text-ink truncate">{name}</p>}
                {email && (
                  <p className="text-[0.7rem] text-ink-faint truncate">{email}</p>
                )}
              </div>
              <MenuLink to="/account" onSelect={() => setOpen(false)}>
                Account
              </MenuLink>
              <MenuButton onClick={handleLogout}>Log out</MenuButton>
            </>
          ) : (
            <>
              <MenuLink to="/register" onSelect={() => setOpen(false)}>
                Register
              </MenuLink>
              <MenuLink to="/login" onSelect={() => setOpen(false)}>
                Log in
              </MenuLink>
            </>
          )}
        </div>
      )}
    </div>
  )
}

function MenuLink({
  to,
  onSelect,
  children,
}: {
  to: string
  onSelect: () => void
  children: React.ReactNode
}) {
  return (
    <Link
      to={to}
      role="menuitem"
      onClick={onSelect}
      className="block px-4 py-2.5 text-sm text-ink-soft hover:text-ink hover:bg-sunken transition-colors border-t border-rule first:border-t-0"
    >
      {children}
    </Link>
  )
}

function MenuButton({
  onClick,
  children,
}: {
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      className="block w-full text-left px-4 py-2.5 text-sm text-ink-soft hover:text-ink hover:bg-sunken transition-colors border-t border-rule cursor-pointer"
    >
      {children}
    </button>
  )
}
