import { useEffect, useState } from 'react'
import { Link, NavLink, Outlet } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { useAuth } from '../auth/useAuth'
import { Seo } from '../components/Seo'
import { api, ApiError } from '../lib/api'

const linkBase = 'text-sm text-ink-soft hover:text-ink transition-colors py-1'
const activeStyle =
  'text-ink relative after:content-[""] after:absolute after:left-0 after:right-0 after:-bottom-0.5 after:h-px after:bg-accent'

type NavItem = { to: string; label: string; roles: string[] }

// Mirrors the server-side RBAC: each section lists the roles the backend will
// actually let through, so the UI only offers what the role can use.
const NAV: NavItem[] = [
  { to: '/admin/products', label: 'Products', roles: ['admin', 'sales'] },
  { to: '/admin/categories', label: 'Categories', roles: ['admin', 'sales'] },
  { to: '/admin/brands', label: 'Brands', roles: ['admin', 'sales'] },
  { to: '/admin/orders', label: 'Orders', roles: ['admin', 'support'] },
  { to: '/admin/reviews', label: 'Reviews', roles: ['admin', 'support'] },
  { to: '/admin/users', label: 'Users', roles: ['admin'] },
  { to: '/admin/audit', label: 'Audit', roles: ['admin'] },
]

type Gate = 'loading' | 'ok' | 'needs2fa' | 'forbidden' | 'unauthed'

export function AdminLayout() {
  const { isAuthed, initializing } = useAuth()
  const [gate, setGate] = useState<Gate>('loading')
  const [role, setRole] = useState<string>('')

  useEffect(() => {
    if (initializing) return
    if (!isAuthed) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setGate('unauthed')
      return
    }
    let cancelled = false
    api
      .adminMe()
      .then((me) => {
        if (cancelled) return
        setRole(me.role)
        setGate('ok')
      })
      .catch((e) => {
        if (cancelled) return
        if (e instanceof ApiError && e.code === '2fa_required') setGate('needs2fa')
        else if (e instanceof ApiError && e.status === 403) setGate('forbidden')
        else if (e instanceof ApiError && e.status === 401) setGate('unauthed')
        else setGate('forbidden')
      })
    return () => {
      cancelled = true
    }
  }, [isAuthed, initializing])

  if (gate === 'loading' || initializing) {
    return (
      <Page width="max-w-3xl">
        <p className="text-sm text-ink-soft">Loading.</p>
      </Page>
    )
  }

  if (gate === 'unauthed') {
    return (
      <Page width="max-w-3xl">
        <Masthead eyebrow="Admin" title="Sign in required." />
        <p className="text-ink-soft">
          You need to be signed in as a staff member to view this area.{' '}
          <Link to="/login" className="text-ink underline underline-offset-4">Log in</Link>.
        </p>
      </Page>
    )
  }

  if (gate === 'forbidden') {
    return (
      <Page width="max-w-3xl">
        <Masthead eyebrow="Admin" title="No access." />
        <p className="text-ink-soft">
          This account does not have staff permissions.{' '}
          <Link to="/" className="text-ink underline underline-offset-4">Back to the shop</Link>.
        </p>
      </Page>
    )
  }

  if (gate === 'needs2fa') {
    return (
      <Page width="max-w-3xl">
        <Masthead eyebrow="Admin" title="Two-factor required." />
        <p className="text-ink-soft mb-6">
          Staff accounts must have two-factor authentication enabled before the
          admin area unlocks. Set it up, then sign in again.
        </p>
        <Link
          to="/account/2fa/setup"
          className="inline-block bg-accent text-on-accent px-5 py-2.5 text-sm hover:bg-accent-soft transition-colors"
        >
          Set up two-factor authentication
        </Link>
      </Page>
    )
  }

  const items = NAV.filter((n) => n.roles.includes(role))

  return (
    <>
      <Seo title="Admin" noindex />
      <div className="max-w-5xl mx-auto px-6 lg:px-10 pt-10 lg:pt-14">
        <div className="flex flex-wrap items-baseline gap-x-6 gap-y-2 border-b border-rule pb-3">
          <span className="uc-tight text-[0.7rem] text-ink-faint mr-2">
            <span>Admin</span>
          </span>
          {items.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              end
              className={({ isActive }) => `${linkBase} ${isActive ? activeStyle : ''}`}
            >
              {n.label}
            </NavLink>
          ))}
        </div>
      </div>
      <Outlet />
    </>
  )
}
