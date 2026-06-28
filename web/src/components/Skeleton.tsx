/* Skeleton.tsx — loading placeholders. A single `.skeleton` base (opacity pulse,
 * disabled under prefers-reduced-motion via index.css) plus layout-matched
 * helpers so the page doesn't jump when real content arrives. Decorative, so
 * aria-hidden; the containing region carries aria-busy. */

export function Skeleton({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`skeleton rounded-md ${className}`} />
}

// ProductCardSkeleton mirrors the PLP product card (square image, brand line,
// title, price) so the grid keeps its shape while loading.
function ProductCardSkeleton() {
  return (
    <article className="flex flex-col gap-3">
      <Skeleton className="aspect-square w-full" />
      <div className="space-y-2">
        <Skeleton className="h-3 w-1/3" />
        <Skeleton className="h-4 w-4/5" />
      </div>
      <div className="mt-auto pt-3">
        <Skeleton className="h-5 w-1/2" />
      </div>
    </article>
  )
}

// ProductGridSkeleton fills the same grid the PLP uses with `count` card
// placeholders.
export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div
      aria-busy="true"
      aria-label="Loading products"
      className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 md:gap-5"
    >
      {Array.from({ length: count }).map((_, i) => (
        <ProductCardSkeleton key={i} />
      ))}
    </div>
  )
}
