/* CampaignBanners.tsx — static marketing tiles for the home page. Copy lives
 * in the consts below; it's editorial demo content, not driven by the API.
 */
import { Link } from 'react-router-dom'
import { Truck } from '../components/icons'

/* Two compact tiles stacked beside the hero carousel on large screens. */
export function HeroSideTiles() {
  return (
    <div className="hidden lg:flex flex-col gap-4">
      <Link
        to="/?sort=discount"
        className="group flex flex-1 flex-col justify-center rounded-xl bg-accent text-on-accent px-6 py-5 transition-shadow hover:shadow-[0_8px_24px_oklch(0.2_0.01_265/0.25)]"
      >
        <span className="uc-tight text-[0.7rem] opacity-90">Outlet</span>
        <span className="font-display text-2xl font-extrabold leading-tight mt-1">
          Last chance deals
        </span>
        <span className="mt-2 text-sm underline underline-offset-4 decoration-on-accent/50 group-hover:decoration-on-accent transition-colors">
          Shop the outlet
        </span>
      </Link>
      <div className="flex flex-1 items-center gap-4 rounded-xl bg-sunken px-6 py-5">
        <Truck size={32} strokeWidth={1.5} aria-hidden className="shrink-0 text-primary" />
        <div>
          <p className="font-display text-lg font-extrabold leading-tight text-ink">
            Free shipping over $50
          </p>
          <p className="mt-1 text-sm text-ink-soft">On standard delivery, every day.</p>
        </div>
      </div>
    </div>
  )
}

/* Mid-page banner duo between the sale band and the product rails. */
export function CampaignBanners() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Link
        to="/?sort=discount"
        className="group flex flex-col justify-center rounded-xl bg-accent text-on-accent px-6 py-8 sm:px-8 transition-shadow hover:shadow-[0_8px_24px_oklch(0.2_0.01_265/0.25)]"
      >
        <span className="uc-tight text-[0.7rem] opacity-90">Outlet</span>
        <span className="font-display text-2xl sm:text-3xl font-extrabold leading-tight mt-1">
          Up to 60% off — while stock lasts
        </span>
        <span className="mt-3 text-sm underline underline-offset-4 decoration-on-accent/50 group-hover:decoration-on-accent transition-colors">
          See all discounts
        </span>
      </Link>
      <Link
        to="/?sort=rating"
        className="group flex flex-col justify-center rounded-xl bg-primary-soft text-on-primary px-6 py-8 sm:px-8 transition-shadow hover:shadow-[0_8px_24px_oklch(0.2_0.01_265/0.25)]"
      >
        <span className="uc-tight text-[0.7rem] opacity-90">Customer favourites</span>
        <span className="font-display text-2xl sm:text-3xl font-extrabold leading-tight mt-1">
          The stuff shoppers rate highest
        </span>
        <span className="mt-3 text-sm underline underline-offset-4 decoration-on-primary/50 group-hover:decoration-on-primary transition-colors">
          Browse top rated
        </span>
      </Link>
    </div>
  )
}
