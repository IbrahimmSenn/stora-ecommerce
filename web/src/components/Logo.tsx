/* Logo.tsx — Stora brand mark + wordmark.
 *
 * Mark: a rounded highlight tile with a bold "S" monogram in the display face.
 * Wordmark: large, bold display type. Used on the header and footer (both on
 * the primary band), so the wordmark inherits on-primary; the tile carries the
 * accent and stays high-contrast against the primary background.
 */
import { Link } from 'react-router-dom'

export function Logo({ className = '' }: { className?: string }) {
  return (
    <Link
      to="/"
      aria-label="Stora, home"
      className={`group inline-flex items-center gap-2.5 shrink-0 text-on-primary ${className}`}
    >
      <span
        aria-hidden
        className="inline-flex h-9 w-9 lg:h-10 lg:w-10 items-center justify-center rounded-xl bg-highlight text-primary font-display font-black text-xl lg:text-2xl leading-none shadow-sm transition-transform group-hover:-translate-y-0.5"
      >
        S
      </span>
      <span className="font-display text-2xl lg:text-3xl font-extrabold tracking-tight leading-none">
        Stora
      </span>
    </Link>
  )
}
