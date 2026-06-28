/* Footer.tsx — site-wide footer.
 *
 * Multi-column link groups on a dark primary band, with the theme toggle and a
 * copyright line. Rendered once in App after <main>.
 */
import { Link } from 'react-router-dom'
import { ThemeToggle } from './ThemeToggle'
import { Logo } from './Logo'

type FooterLink = { label: string; to: string }

const columns: { heading: string; links: FooterLink[] }[] = [
  {
    heading: 'Shop',
    links: [
      { label: 'All products', to: '/' },
      { label: 'Cart', to: '/cart' },
      { label: 'Orders', to: '/orders' },
    ],
  },
  {
    heading: 'Account',
    links: [
      { label: 'Sign in', to: '/login' },
      { label: 'Register', to: '/register' },
      { label: 'Your account', to: '/account' },
    ],
  },
  {
    heading: 'Company',
    links: [
      { label: 'About', to: '/about' },
      { label: 'Contact', to: '/contact' },
    ],
  },
]

export function Footer() {
  return (
    <footer className="mt-16 bg-primary text-on-primary">
      <div className="max-w-7xl mx-auto px-4 lg:px-8 py-12 grid grid-cols-2 md:grid-cols-4 gap-8">
        <div className="col-span-2 md:col-span-1">
          <Logo />
          <p className="mt-3 text-sm text-on-primary/70 max-w-[28ch]">
            Shop electronics, furniture, beauty, shoes and more — with fast,
            secure checkout.
          </p>
        </div>

        {columns.map((col) => (
          <nav key={col.heading} aria-label={col.heading}>
            <h2 className="text-xs uppercase tracking-wide text-on-primary/60 mb-3">
              {col.heading}
            </h2>
            <ul className="space-y-2">
              {col.links.map((l) => (
                <li key={l.to + l.label}>
                  <Link
                    to={l.to}
                    className="text-sm text-on-primary/85 hover:text-on-primary transition-colors"
                  >
                    {l.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        ))}
      </div>

      <div className="border-t border-on-primary/15">
        <div className="max-w-7xl mx-auto px-4 lg:px-8 py-5 flex items-center justify-between gap-4">
          <p className="text-xs text-on-primary/60">
            © {new Date().getFullYear()} Stora. Demo project.
          </p>
          <ThemeToggle onDark />
        </div>
      </div>
    </footer>
  )
}
