/* HomeSections.tsx — the campaign landing content rendered above the "All
 * products" grid on the pristine home route. Owns its own data fetching; every
 * rail hides itself when its data is missing so the home page never shows an
 * error for a decorative section.
 */
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import type { Brand, Category, ProductListItem } from '../lib/api'
import { PromoCarousel } from './PromoCarousel'
import { MegaSale } from './MegaSale'
import { CategoryTiles } from './CategoryTiles'
import { CampaignBanners, HeroSideTiles } from './CampaignBanners'
import { Countdown } from './Countdown'
import { ProductCard } from './ProductCard'
import { Rail } from '../components/Rail'
import { Reveal } from '../lib/motion'

type HomeData = {
  sale: ProductListItem[]
  categories: Category[]
  bestsellers: ProductListItem[]
  topRated: ProductListItem[]
  forYou: ProductListItem[]
  brands: Brand[]
}

const EMPTY: HomeData = {
  sale: [],
  categories: [],
  bestsellers: [],
  topRated: [],
  forYou: [],
  brands: [],
}

export function HomeSections({
  busyId,
  onAdd,
}: {
  busyId: string | null
  onAdd: (p: ProductListItem) => void
}) {
  const [data, setData] = useState<HomeData>(EMPTY)

  useEffect(() => {
    let cancelled = false
    Promise.allSettled([
      api.listProducts({ onSale: true, sort: 'discount', pageSize: 20 }),
      api.listCategories(),
      api.listProducts({ sort: 'bestseller', pageSize: 10 }),
      api.listProducts({ sort: 'rating', pageSize: 10 }),
      api.recommendations(10),
      api.listBrands(),
    ]).then(([sale, cats, best, rated, recs, brands]) => {
      if (cancelled) return
      setData({
        sale: sale.status === 'fulfilled' ? sale.value.products : [],
        categories: cats.status === 'fulfilled' ? cats.value : [],
        bestsellers: best.status === 'fulfilled' ? best.value.products : [],
        topRated: rated.status === 'fulfilled' ? rated.value.products : [],
        forYou: recs.status === 'fulfilled' ? (recs.value.items ?? []) : [],
        brands: brands.status === 'fulfilled' ? brands.value : [],
      })
    })
    return () => {
      cancelled = true
    }
  }, [])

  const { sale, categories, bestsellers, topRated, forYou, brands } = data

  return (
    <Reveal className="flex flex-col gap-10 lg:gap-12 mb-12" stagger={70}>
      {sale.length > 0 && (
        <div className="grid gap-4 lg:grid-cols-[minmax(0,2.4fr)_minmax(0,1fr)]">
          <PromoCarousel products={sale.slice(0, 6)} />
          <HeroSideTiles />
        </div>
      )}

      {categories.length > 0 && (
        <div>
          <CategoryTiles categories={categories} />
        </div>
      )}

      {sale.length > 0 && (
        <div>
          <MegaSale products={sale} eyebrow={<Countdown />} busyId={busyId} onAdd={onAdd} />
        </div>
      )}

      <div>
        <CampaignBanners />
      </div>

      {bestsellers.length > 0 && (
        <div>
          <Rail
            id="bestsellers"
            title="Best sellers"
            eyebrow="What everyone's buying right now."
            action={<SeeAll to="/?sort=bestseller" label="See all best sellers" />}
          >
            {bestsellers.map((p, i) => (
              <li key={p.id} className="shrink-0 snap-start">
                <ProductCard
                  product={p}
                  variant="rail"
                  badge={i < 3 ? 'bestseller' : undefined}
                  busy={busyId === p.id}
                  onAdd={() => onAdd(p)}
                />
              </li>
            ))}
          </Rail>
        </div>
      )}

      {topRated.length > 0 && (
        <div>
          <Rail
            id="top-rated"
            tone="soft"
            title="Top rated"
            eyebrow="Loved by the people who bought them."
            action={<SeeAll to="/?sort=rating" label="See all top rated" />}
          >
            {topRated.map((p) => (
              <li key={p.id} className="shrink-0 snap-start">
                <ProductCard product={p} variant="rail" busy={busyId === p.id} onAdd={() => onAdd(p)} />
              </li>
            ))}
          </Rail>
        </div>
      )}

      {forYou.length > 0 && (
        <div>
          <Rail id="for-you" tone="soft" title="Picked for you" eyebrow="Based on what you've looked at.">
            {forYou.map((p) => (
              <li key={p.id} className="shrink-0 snap-start">
                <ProductCard product={p} variant="rail" busy={busyId === p.id} onAdd={() => onAdd(p)} />
              </li>
            ))}
          </Rail>
        </div>
      )}

      {brands.length > 0 && (
        <section aria-labelledby="brands-heading">
          <h2 id="brands-heading" className="uc-tight text-[0.7rem] text-ink-faint mb-3">
            Shop by brand
          </h2>
          <ul className="flex flex-wrap gap-2">
            {brands.slice(0, 18).map((b) => (
              <li key={b.id}>
                <Link
                  to={`/?brand=${b.id}`}
                  className="inline-block rounded-full border border-rule px-3 py-1.5 text-sm text-ink-soft hover:border-primary hover:text-primary transition-colors"
                >
                  {b.name}
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </Reveal>
  )
}

function SeeAll({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="text-sm text-ink-soft underline underline-offset-4 decoration-rule-strong hover:text-primary hover:decoration-primary transition-colors whitespace-nowrap"
    >
      <span aria-hidden>See all</span>
      <span className="sr-only">{label}</span>
    </Link>
  )
}
