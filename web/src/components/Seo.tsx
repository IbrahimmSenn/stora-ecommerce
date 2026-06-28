/* Seo.tsx — per-page <title> and meta tags.
 *
 * React 19 hoists <title>/<meta>/<link> rendered anywhere in the tree into
 * <head>, so no helmet dependency is needed. Keep the full title under ~60
 * chars (brand suffix included) and descriptions around 150–160 chars.
 */

const SITE = 'Stora'

type SeoProps = {
  /** Page-specific title; the brand is appended. Omit to show just the brand. */
  title?: string
  description?: string
  /** Discourage indexing (admin / account areas). */
  noindex?: boolean
}

export function Seo({ title, description, noindex }: SeoProps) {
  const full = title ? `${title} · ${SITE}` : `${SITE} — Online Shopping`
  return (
    <>
      <title>{full}</title>
      {description && <meta name="description" content={description} />}
      {noindex && <meta name="robots" content="noindex,nofollow" />}
    </>
  )
}
