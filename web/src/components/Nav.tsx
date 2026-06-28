/* Nav.tsx — marketplace header.
 *
 * Two tiers on a saturated primary bar: top row is the STORA wordmark, a
 * prominent search, and the account / cart controls; beneath it a category
 * strip (CategoryBar). Sticky so it stays in reach while scrolling. The
 * hamburger opens the side panel (categories, nav links, theme toggle) on
 * narrow screens.
 */
import { useRef } from 'react'
import { NavLink } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useSidePanel } from './useSidePanel'
import { Menu } from './icons'
import { Logo } from './Logo'
import { NavCart } from './NavCart'
import { NavProfile } from './NavProfile'
import { NavSearch } from './NavSearch'
import { CategoryBar } from './CategoryBar'

export function Nav() {
  const { role } = useAuth()
  const { isOpen: sidePanelOpen, toggle: toggleSidePanel } = useSidePanel()
  const hamburgerRef = useRef<HTMLButtonElement | null>(null)

  return (
    <header className="sticky top-0 z-30 bg-primary text-on-primary">
      <div className="max-w-7xl mx-auto px-4 lg:px-8 py-3 flex items-center gap-4 lg:gap-6">
        <button
          ref={hamburgerRef}
          type="button"
          onClick={() => toggleSidePanel(hamburgerRef.current)}
          aria-haspopup="dialog"
          aria-expanded={sidePanelOpen}
          aria-label="Open menu"
          className="inline-flex h-11 w-11 items-center justify-center shrink-0 rounded-md transition-colors cursor-pointer text-on-primary hover:bg-on-primary/15"
        >
          <Menu size={28} strokeWidth={2.25} aria-hidden />
        </button>

        <Logo />

        <div className="flex-1 min-w-0 max-w-2xl">
          <NavSearch prominent />
        </div>

        {role === 'admin' && (
          <NavLink
            to="/admin/products"
            className={({ isActive }) =>
              `hidden lg:inline text-sm transition-colors ${
                isActive
                  ? 'text-on-primary'
                  : 'text-on-primary/80 hover:text-on-primary'
              }`
            }
          >
            Admin
          </NavLink>
        )}

        <div className="flex items-center gap-1 shrink-0">
          <NavProfile onDark />
          <NavCart onDark />
        </div>
      </div>

      <CategoryBar />
    </header>
  )
}
