/* CategoryBar.tsx — horizontal category strip under the header.
 *
 * Loads top-level categories once and renders them as a scrollable row of
 * links, Amazon/Gigantti style. "All" returns to the full catalogue. Sits on a
 * slightly darker shade of the header (primary-soft) so it reads as a second
 * tier of the same bar.
 */
import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { api } from '../lib/api'
import type { Category } from '../lib/api'

export function CategoryBar() {
  const [categories, setCategories] = useState<Category[]>([])

  useEffect(() => {
    let cancelled = false
    api
      .listCategories()
      .then((cats) => {
        if (!cancelled) setCategories(cats)
      })
      .catch(() => {
        if (!cancelled) setCategories([])
      })
    return () => {
      cancelled = true
    }
  }, [])

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `shrink-0 whitespace-nowrap px-3 py-1.5 rounded-full text-sm transition-colors ${
      isActive
        ? 'bg-on-primary text-primary font-medium'
        : 'text-on-primary/85 hover:bg-on-primary/15 hover:text-on-primary'
    }`

  return (
    <nav aria-label="Categories" className="bg-primary-soft">
      <div className="max-w-7xl mx-auto px-4 lg:px-8">
        <ul className="flex items-center gap-1 overflow-x-auto py-2 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
          <li>
            <NavLink to="/" end className={linkClass}>
              All
            </NavLink>
          </li>
          {categories.map((c) => (
            <li key={c.id}>
              <NavLink to={`/shop/${c.slug}`} className={linkClass}>
                {c.name}
              </NavLink>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
