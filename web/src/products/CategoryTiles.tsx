/* CategoryTiles.tsx — "Shop by category" image tile grid on the home page.
 * Tiles link to /shop/:slug; a category without an image gets a name-only tile.
 */
import { Link } from 'react-router-dom'
import type { Category } from '../lib/api'

export function CategoryTiles({ categories }: { categories: Category[] }) {
  const topLevel = categories.filter((c) => !c.parent_id)
  if (topLevel.length === 0) return null

  return (
    <section aria-labelledby="category-tiles-heading">
      <h2
        id="category-tiles-heading"
        className="font-display text-2xl md:text-3xl font-extrabold tracking-tight text-ink mb-4"
      >
        Shop by category
      </h2>
      <ul className="grid grid-cols-3 sm:grid-cols-5 gap-3 md:gap-4">
        {topLevel.map((c) => (
          <li key={c.id}>
            <Link
              to={`/shop/${c.slug}`}
              className="group flex flex-col items-center gap-2 rounded-lg bg-sunken p-3 pt-4 transition-shadow hover:shadow-[0_6px_20px_oklch(0.2_0.01_265/0.12)]"
            >
              <div className="h-20 w-20 sm:h-24 sm:w-24 flex items-center justify-center">
                {c.image_url ? (
                  // Decorative: the tile's text names the category.
                  <img
                    src={c.image_url}
                    alt=""
                    loading="lazy"
                    className="max-h-full max-w-full object-contain transition-transform duration-300 group-hover:scale-[1.06]"
                  />
                ) : (
                  <span aria-hidden className="font-display text-3xl font-extrabold text-ink-faint">
                    {c.name.charAt(0)}
                  </span>
                )}
              </div>
              <span className="text-sm font-medium text-ink text-center leading-tight group-hover:text-primary transition-colors">
                {c.name}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
