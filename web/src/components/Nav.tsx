/* Nav.tsx — top navigation.
 *
 * STORA wordmark (home link) — search input — cart icon (badge) — profile
 * icon (dropdown) — hamburger (opens side panel). Admin link still surfaces
 * when role=admin so the panel doesn't have to host it. Theme toggle has
 * moved into the side panel.
 */
import { useRef } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useSidePanel } from './useSidePanel'
import { Menu } from './icons'
import { NavCart } from './NavCart'
import { NavProfile } from './NavProfile'
import { NavSearch } from './NavSearch'

export function Nav() {
  const { role } = useAuth()
  const { isOpen: sidePanelOpen, toggle: toggleSidePanel } = useSidePanel()
  const hamburgerRef = useRef<HTMLButtonElement | null>(null)

  return (
    <nav className="border-b border-rule">
      <div className="max-w-6xl mx-auto px-6 lg:px-10 py-5 flex items-center gap-8">
        <Link
          to="/"
          aria-label="Stora, home"
          className="font-display text-[0.95rem] uppercase tracking-[0.32em] font-bold shrink-0"
        >
          STORA
        </Link>

        <div className="hidden md:block">
          <NavSearch />
        </div>

        {role === 'admin' && (
          <NavLink
            to="/admin/products"
            className={({ isActive }) =>
              `hidden lg:inline text-sm transition-colors ${
                isActive ? 'text-ink' : 'text-ink-soft hover:text-ink'
              }`
            }
          >
            Admin
          </NavLink>
        )}

        <div className="ml-auto flex items-center gap-1">
          <NavCart />
          <NavProfile />
          <button
            ref={hamburgerRef}
            type="button"
            onClick={() => toggleSidePanel(hamburgerRef.current)}
            aria-haspopup="dialog"
            aria-expanded={sidePanelOpen}
            aria-label="Open menu"
            className={`inline-flex h-9 w-9 items-center justify-center transition-colors cursor-pointer ${
              sidePanelOpen ? 'text-ink' : 'text-ink-soft hover:text-ink'
            }`}
          >
            <Menu size={18} strokeWidth={1.5} aria-hidden />
          </button>
        </div>
      </div>
    </nav>
  )
}
