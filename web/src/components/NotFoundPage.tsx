import { Link } from 'react-router-dom'
import { Page } from './Page'
import { Masthead } from './Masthead'
import { Seo } from './Seo'

export function NotFoundPage() {
  return (
    <Page width="max-w-3xl">
      <Seo title="Page not found" noindex />
      <Masthead eyebrow="404" title="Page not found" />
      <p className="text-ink-soft leading-relaxed mb-8 max-w-prose">
        The page you’re looking for doesn’t exist. It may have been moved or
        removed, or the link may be incorrect.
      </p>
      <div className="flex flex-wrap gap-x-8 gap-y-2 text-sm">
        <Link
          to="/"
          className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          Continue shopping
        </Link>
        <Link
          to="/cart"
          className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors"
        >
          View your cart
        </Link>
      </div>
    </Page>
  )
}
