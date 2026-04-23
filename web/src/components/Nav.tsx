import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useCart } from '../cart/useCart'

export function Nav() {
  const { isAuthed, email, logout } = useAuth()
  const { itemCount, refresh } = useCart()
  const navigate = useNavigate()

  async function handleLogout() {
    logout()
    await refresh()
    navigate('/')
  }

  return (
    <nav className="border-b px-6 py-3 flex items-center gap-6">
      <Link to="/" className="font-semibold">
        i-love-shopping
      </Link>
      <div className="flex items-center gap-4 text-sm">
        <NavLink
          to="/"
          end
          className={({ isActive }) => (isActive ? 'underline' : '')}
        >
          Shop
        </NavLink>
        <NavLink
          to="/cart"
          className={({ isActive }) => (isActive ? 'underline' : '')}
        >
          Cart {itemCount > 0 && <span className="tabular-nums">({itemCount})</span>}
        </NavLink>
      </div>
      <div className="ml-auto text-sm flex items-center gap-3">
        {isAuthed ? (
          <>
            <span className="text-gray-600">{email}</span>
            <button
              type="button"
              onClick={handleLogout}
              className="underline"
            >
              Log out
            </button>
          </>
        ) : (
          <NavLink
            to="/login"
            className={({ isActive }) => (isActive ? 'underline' : '')}
          >
            Log in
          </NavLink>
        )}
      </div>
    </nav>
  )
}
