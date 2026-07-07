/* Seo.tsx — per-page <title>, meta, Open Graph / Twitter Card and canonical.
 *
 * React 19 hoists <title>/<meta>/<link> rendered anywhere in the tree into
 * <head>, so no helmet dependency is needed. Keep the full title under ~60
 * chars (brand suffix included) and descriptions around 150–160 chars.
 */

import { useLocation } from 'react-router-dom'

const SITE = 'Stora'

type SeoProps = {
  /** Page-specific title; the brand is appended. Omit to show just the brand. */
  title?: string
  description?: string
  /** Discourage indexing (admin / account areas). Also suppresses canonical/OG. */
  noindex?: boolean
  /** Share image URL (absolute or site-relative) for link unfurls. */
  image?: string | null
  /** Open Graph type; PDPs pass "product". */
  ogType?: 'website' | 'product'
}

function absolute(url: string): string {
  return url.startsWith('http') ? url : window.location.origin + url
}

export function Seo({ title, description, noindex, image, ogType = 'website' }: SeoProps) {
  const { pathname } = useLocation()
  const full = title ? `${title} · ${SITE}` : `${SITE} — Online Shopping`

  if (noindex) {
    return (
      <>
        <title>{full}</title>
        {description && <meta name="description" content={description} />}
        <meta name="robots" content="noindex,nofollow" />
      </>
    )
  }

  const canonical = window.location.origin + pathname
  const img = image ? absolute(image) : null
  return (
    <>
      <title>{full}</title>
      {description && <meta name="description" content={description} />}
      <link rel="canonical" href={canonical} />
      <meta property="og:site_name" content={SITE} />
      <meta property="og:type" content={ogType} />
      <meta property="og:title" content={full} />
      {description && <meta property="og:description" content={description} />}
      <meta property="og:url" content={canonical} />
      {img && <meta property="og:image" content={img} />}
      <meta name="twitter:card" content={img ? 'summary_large_image' : 'summary'} />
      <meta name="twitter:title" content={full} />
      {description && <meta name="twitter:description" content={description} />}
      {img && <meta name="twitter:image" content={img} />}
    </>
  )
}
