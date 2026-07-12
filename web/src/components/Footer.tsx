/* Footer.tsx — site-wide footer.
 *
 * Trust strip + newsletter band above multi-column link groups on a dark
 * primary band, then payment methods, social links, theme toggle, and the
 * copyright line. Rendered once in App after <main>.
 */
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ThemeToggle } from './ThemeToggle'
import { Logo } from './Logo'
import { TrustRow } from './TrustRow'
import { PaymentIcons } from './PaymentIcons'
import { useToast } from './useToast'

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

const socials: { label: string; href: string; path: string }[] = [
  {
    label: 'X',
    href: 'https://x.com',
    path: 'M18.9 2H22l-6.8 7.8L23.2 22h-6.3l-4.9-6.5L6.4 22H3.3l7.3-8.3L2.8 2h6.4l4.4 5.9L18.9 2Zm-1.1 18h1.7L7.1 3.9H5.3L17.8 20Z',
  },
  {
    label: 'Instagram',
    href: 'https://instagram.com',
    path: 'M12 2.2c3.2 0 3.6 0 4.8.1 1.2.1 1.9.2 2.3.4.6.2 1 .5 1.4.9.4.4.7.9.9 1.4.2.4.4 1.1.4 2.3.1 1.3.1 1.6.1 4.8s0 3.6-.1 4.8c-.1 1.2-.2 1.9-.4 2.3-.2.6-.5 1-.9 1.4-.4.4-.9.7-1.4.9-.4.2-1.1.4-2.3.4-1.3.1-1.6.1-4.8.1s-3.6 0-4.8-.1c-1.2-.1-1.9-.2-2.3-.4-.6-.2-1-.5-1.4-.9-.4-.4-.7-.9-.9-1.4-.2-.4-.4-1.1-.4-2.3-.1-1.3-.1-1.6-.1-4.8s0-3.6.1-4.8c.1-1.2.2-1.9.4-2.3.2-.6.5-1 .9-1.4.4-.4.9-.7 1.4-.9.4-.2 1.1-.4 2.3-.4C8.4 2.2 8.8 2.2 12 2.2Zm0 1.8c-3.1 0-3.5 0-4.7.1-1.1.1-1.7.2-2.1.4-.5.2-.9.4-1.2.8-.4.4-.6.7-.8 1.2-.2.4-.3 1-.4 2.1-.1 1.2-.1 1.6-.1 4.7s0 3.5.1 4.7c.1 1.1.2 1.7.4 2.1.2.5.4.9.8 1.2.4.4.7.6 1.2.8.4.2 1 .3 2.1.4 1.2.1 1.6.1 4.7.1s3.5 0 4.7-.1c1.1-.1 1.7-.2 2.1-.4.5-.2.9-.4 1.2-.8.4-.4.6-.7.8-1.2.2-.4.3-1 .4-2.1.1-1.2.1-1.6.1-4.7s0-3.5-.1-4.7c-.1-1.1-.2-1.7-.4-2.1-.2-.5-.4-.9-.8-1.2-.4-.4-.7-.6-1.2-.8-.4-.2-1-.3-2.1-.4-1.2-.1-1.6-.1-4.7-.1Zm0 3.1a4.9 4.9 0 1 1 0 9.8 4.9 4.9 0 0 1 0-9.8Zm0 1.8a3.1 3.1 0 1 0 0 6.2 3.1 3.1 0 0 0 0-6.2Zm5.1-3.1a1.15 1.15 0 1 1 0 2.3 1.15 1.15 0 0 1 0-2.3Z',
  },
  {
    label: 'YouTube',
    href: 'https://youtube.com',
    path: 'M23 7.2s-.2-1.6-.9-2.3c-.9-.9-1.9-.9-2.3-1C16.6 3.6 12 3.6 12 3.6s-4.6 0-7.8.3c-.4.1-1.4.1-2.3 1-.7.7-.9 2.3-.9 2.3S.8 9.1.8 11v1.8c0 1.9.2 3.8.2 3.8s.2 1.6.9 2.3c.9.9 2 .9 2.5 1 1.8.2 7.6.3 7.6.3s4.6 0 7.8-.3c.4-.1 1.4-.1 2.3-1 .7-.7.9-2.3.9-2.3s.2-1.9.2-3.8V11c0-1.9-.2-3.8-.2-3.8ZM9.7 15.1V8.6l6.1 3.3-6.1 3.2Z',
  },
]

function NewsletterForm() {
  const { show: showToast } = useToast()
  const [email, setEmail] = useState('')

  // Demo-only: no mailing list backend exists, so submitting just confirms
  // via toast and clears the field.
  function submit(e: React.FormEvent) {
    e.preventDefault()
    setEmail('')
    showToast('Thanks! (Demo — no emails are sent.)')
  }

  return (
    <form onSubmit={submit} className="w-full max-w-md">
      <label htmlFor="footer-newsletter" className="block text-sm font-semibold text-on-primary">
        Get deals first
      </label>
      <p className="mt-1 text-xs text-on-primary/70">
        Sale alerts and new arrivals, straight to your inbox.
      </p>
      <div className="mt-3 flex items-stretch rounded-md overflow-hidden ring-1 ring-on-primary/25 focus-within:ring-2 focus-within:ring-highlight transition-shadow">
        <input
          id="footer-newsletter"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          className="flex-1 min-w-0 bg-on-primary/10 px-3 py-2 text-sm text-on-primary placeholder:text-on-primary/50 focus:outline-none"
        />
        <button
          type="submit"
          className="shrink-0 bg-highlight px-4 text-sm font-bold text-highlight-ink hover:brightness-95 transition cursor-pointer"
        >
          Sign up
        </button>
      </div>
    </form>
  )
}

export function Footer() {
  return (
    <footer className="mt-16 bg-primary text-on-primary">
      <div className="max-w-7xl mx-auto px-4 lg:px-8 py-8 border-b border-on-primary/15">
        <TrustRow onDark />
      </div>

      <div className="max-w-7xl mx-auto px-4 lg:px-8 py-12 grid grid-cols-2 md:grid-cols-5 gap-8">
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

        <div className="col-span-2 md:col-span-1">
          <NewsletterForm />
        </div>
      </div>

      <div className="border-t border-on-primary/15">
        <div className="max-w-7xl mx-auto px-4 lg:px-8 py-5 flex flex-wrap items-center justify-between gap-4">
          <PaymentIcons />
          <ul className="flex items-center gap-2" aria-label="Social media">
            {socials.map((s) => (
              <li key={s.label}>
                <a
                  href={s.href}
                  target="_blank"
                  rel="noreferrer"
                  aria-label={`Stora on ${s.label}`}
                  className="inline-flex h-8 w-8 items-center justify-center rounded-full text-on-primary/70 hover:text-on-primary hover:bg-on-primary/10 transition-colors"
                >
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden>
                    <path d={s.path} />
                  </svg>
                </a>
              </li>
            ))}
          </ul>
        </div>
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
