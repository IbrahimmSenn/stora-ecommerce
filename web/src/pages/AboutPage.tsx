import { Link } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Seo } from '../components/Seo'

const team = [
  { name: 'Mara Ellison', role: 'Founder & buyer' },
  { name: 'Tomas Reyes', role: 'Operations' },
  { name: 'Aiko Tanaka', role: 'Design & web' },
]

const socials = [
  { label: 'Instagram', href: 'https://instagram.com' },
  { label: 'Bluesky', href: 'https://bsky.app' },
  { label: 'GitHub', href: 'https://github.com' },
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
        description="Learn about Stora, an online store offering electronics, furniture, beauty, shoes and more — with customer reviews, clear pricing, and fast, secure checkout."
      />
      <Masthead
        eyebrow="About"
        title="About Stora"
        caption="Stora is an online store offering a wide range of products across electronics, home, beauty, fashion and more."
      />

      <Section title="Our mission">
        <p className="text-ink-soft leading-relaxed max-w-prose">
          Our goal is to make online shopping simple and trustworthy: clear
          pricing, genuine customer reviews, secure payments, and reliable
          delivery. We want every order to arrive as expected, and every
          question to get a helpful answer.
        </p>
      </Section>

      <Section title="What we sell">
        <p className="text-ink-soft leading-relaxed max-w-prose">
          A broad catalogue spanning electronics, furniture, home, beauty,
          fashion, footwear and more. Each product page shows detailed
          information, multiple images, and ratings from verified buyers.
        </p>
      </Section>

      <Section title="The team">
        <ul className="grid grid-cols-1 sm:grid-cols-3 gap-6">
          {team.map((m) => (
            <li key={m.name}>
              <p className="text-ink font-medium">{m.name}</p>
              <p className="text-sm text-ink-faint">{m.role}</p>
            </li>
          ))}
        </ul>
      </Section>

      <Section title="Find us">
        <ul className="flex flex-wrap gap-x-8 gap-y-2">
          {socials.map((s) => (
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
              Contact us
            </Link>
          </li>
        </ul>
      </Section>
    </Page>
  )
}
