import { Link } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Seo } from '../components/Seo'

const links = [
  { label: 'GitHub — source code', href: 'https://github.com/IbrahimmSenn/stora-ecommerce' },
  { label: 'm.ibrahimsenn@gmail.com', href: 'mailto:m.ibrahimsenn@gmail.com' },
]

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-t border-rule pt-8 mb-14">
      <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-5">{title}</h2>
      {children}
    </section>
  )
}

export function AboutPage() {
  return (
    <Page width="max-w-3xl">
      <Seo
        title="About Stora"
        description="Stora is a portfolio project by Ibrahim Sen — a full-stack e-commerce platform built with Go, PostgreSQL, RabbitMQ, React, and Stripe. Nothing is sold; payments run in test mode."
      />
      <Masthead
        eyebrow="About"
        title="About Stora"
        caption="Stora is a portfolio project — a full working e-commerce platform built to demonstrate real-world engineering, not to sell real products."
      />

      <Section title="What this is">
        <p className="text-ink-soft leading-relaxed max-w-prose">
          Everything here works end to end: browsing, cart, checkout, order
          history, reviews, and an admin area — but the shop is a demonstration.
          Payments run in Stripe test mode, the catalogue is seed data, and no
          real orders are shipped.
        </p>
      </Section>

      <Section title="How it's built">
        <p className="text-ink-soft leading-relaxed max-w-prose">
          A Go backend (PostgreSQL, RabbitMQ, Stripe) with a React + TypeScript
          frontend, encrypted PII at rest, two-factor admin authentication, a
          full observability stack, and a CI/CD pipeline. The complete source
          code and documentation are on GitHub.
        </p>
      </Section>

      <Section title="Built by">
        <p className="text-ink font-medium">Ibrahim Sen</p>
        <p className="text-sm text-ink-faint">Full-stack developer</p>
      </Section>

      <Section title="Find me">
        <ul className="flex flex-wrap gap-x-8 gap-y-2">
          {links.map((s) => (
            <li key={s.label}>
              <a
                href={s.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
              >
                {s.label}
              </a>
            </li>
          ))}
          <li>
            <Link
              to="/contact"
              className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
            >
              Contact form
            </Link>
          </li>
        </ul>
      </Section>
    </Page>
  )
}
