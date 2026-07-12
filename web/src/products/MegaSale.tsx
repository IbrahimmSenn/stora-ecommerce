/* MegaSale.tsx — loud primary-colored band of the day's biggest discounts,
 * rendered as a scroll-snapping rail of white deal cards.
 */
import type { ReactNode } from 'react'
import type { ProductListItem } from '../lib/api'
import { Rail } from '../components/Rail'
import { ProductCard } from './ProductCard'

export function MegaSale({
  products,
  eyebrow,
  busyId,
  onAdd,
}: {
  products: ProductListItem[]
  /** Extra header line, e.g. the deal countdown. */
  eyebrow?: ReactNode
  busyId?: string | null
  onAdd?: (p: ProductListItem) => void
}) {
  if (products.length === 0) return null

  return (
    <Rail
      id="mega-sale"
      tone="loud"
      title="Mega sale"
      eyebrow={eyebrow ?? "Today's biggest discounts, while stock lasts."}
    >
      {products.map((p) => (
        <li key={p.id} className="shrink-0 snap-start">
          <ProductCard
            product={p}
            variant="rail"
            busy={busyId === p.id}
            onAdd={onAdd ? () => onAdd(p) : undefined}
          />
        </li>
      ))}
    </Rail>
  )
}
