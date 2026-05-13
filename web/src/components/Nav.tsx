import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useCart } from '../cart/useCart'
import { useReducedMotion } from '../lib/motion'
import { ThemeToggle } from './ThemeToggle'

const linkBase =
  'text-sm text-ink-soft hover:text-ink transition-colors py-1'

const activeStyle =
  'text-ink relative after:content-[""] after:absolute after:left-0 after:right-0 after:-bottom-0.5 after:h-px after:bg-accent'

export function Nav() {
  const { isAuthed, email, role, logout } = useAuth()
  const { itemCount, refresh } = useCart()
  const navigate = useNavigate()

  async function handleLogout() {
    logout()
    await refresh()
    navigate('/')
  }

  // Cart count pulse — when itemCount changes (add/remove), give the chip a
  // short scale beat so the persistent nav signal links to the cart action.
  const reduced = useReducedMotion()
  const [pulse, setPulse] = useState(false)
  const prevCount = useRef(itemCount)
  useEffect(() => {
    if (reduced) {
      prevCount.current = itemCount
      return
    }
    if (itemCount !== prevCount.current) {
      prevCount.current = itemCount
      setPulse(true)
      const t = window.setTimeout(() => setPulse(false), 240)
      return () => window.clearTimeout(t)
    }
  }, [itemCount, reduced])

  return (
    <nav className="border-b border-rule">
      <div className="max-w-6xl mx-auto px-6 lg:px-10 py-5 flex items-baseline gap-10">
        <Link
          to="/"
          className="font-display text-[1.05rem] tracking-tight"
          style={{ fontVariationSettings: '"wght" 600, "opsz" 24' }}
        >
          i-love-shopping
        </Link>

        <div className="flex items-baseline gap-6">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Shop
          </NavLink>
          <NavLink
            to="/cart"
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Cart{' '}
            {itemCount > 0 && (
              <span
                className="tnum text-ink-faint inline-block"
                style={{
                  transform: pulse ? 'scale(1.08)' : 'scale(1)',
                  transition: reduced
                    ? 'none'
                    : 'transform 240ms var(--ease-out-quart)',
                  transformOrigin: 'left center',
                }}
              >
                — {itemCount}
              </span>
            )}
          </NavLink>
          <NavLink
            to="/orders"
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Orders
          </NavLink>
          {role === 'admin' && (
            <NavLink
              to="/admin/products"
              className={({ isActive }) =>
                `${linkBase} ${isActive ? activeStyle : ''}`
              }
            >
              Admin
            </NavLink>
          )}
        </div>

        <div className="ml-auto flex items-baseline gap-6">
          {isAuthed ? (
            <>
              <NavLink
                to="/account"
                className={({ isActive }) =>
                  `${linkBase} ${isActive ? activeStyle : ''}`
                }
              >
                <span className="text-ink-faint">Account</span>{' '}
                <span className="text-ink">{email}</span>
              </NavLink>
              <button
                type="button"
                onClick={handleLogout}
                className={`${linkBase} cursor-pointer`}
              >
                Log out
              </button>
            </>
          ) : (
            <>
              <NavLink
                to="/register"
                className={({ isActive }) =>
                  `${linkBase} ${isActive ? activeStyle : ''}`
                }
              >
                Register
              </NavLink>
              <NavLink
                to="/login"
                className={({ isActive }) =>
                  `${linkBase} ${isActive ? activeStyle : ''}`
                }
              >
                Log in
              </NavLink>
            </>
          )}
          <ThemeToggle />
        </div>
      </div>
    </nav>
  )
}
