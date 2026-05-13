import { Link, NavLink, Outlet } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useAuth } from '../auth/useAuth'

const linkBase = 'text-sm text-ink-soft hover:text-ink transition-colors py-1'

const activeStyle =
  'text-ink relative after:content-[""] after:absolute after:left-0 after:right-0 after:-bottom-0.5 after:h-px after:bg-accent'

export function AdminLayout() {
  const { role, isAuthed } = useAuth()

  if (!isAuthed) {
    return (
      <Page width="max-w-3xl">
        <Masthead number="00" eyebrow="Admin" title="Sign in required." />
        <p className="text-ink-soft">
          You need to be signed in as an administrator to view this area.{' '}
          <Link to="/login" className="text-ink underline underline-offset-4">
            Log in
          </Link>
          .
        </p>
      </Page>
    )
  }

  if (role !== 'admin') {
    return (
      <Page width="max-w-3xl">
        <Masthead number="00" eyebrow="Admin" title="Admin access required." />
        <p className="text-ink-soft">
          This account does not have administrator permissions.{' '}
          <Link to="/" className="text-ink underline underline-offset-4">
            Back to the shop
          </Link>
          .
        </p>
      </Page>
    )
  }

  return (
    <>
      <div className="max-w-5xl mx-auto px-6 lg:px-10 pt-10 lg:pt-14">
        <div className="flex items-baseline gap-6 border-b border-rule pb-3">
          <span className="uc-tight text-[0.7rem] text-ink-faint mr-2">
            <span className="tnum">03</span>
            <span aria-hidden className="text-rule-strong mx-2">
              /
            </span>
            <span>Admin</span>
          </span>
          <NavLink
            to="/admin/products"
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Products
          </NavLink>
          <NavLink
            to="/admin/categories"
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Categories
          </NavLink>
          <NavLink
            to="/admin/brands"
            className={({ isActive }) =>
              `${linkBase} ${isActive ? activeStyle : ''}`
            }
          >
            Brands
          </NavLink>
        </div>
      </div>
      <Outlet />
    </>
  )
}
