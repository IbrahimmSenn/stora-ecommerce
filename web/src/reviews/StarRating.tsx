/* StarRating.tsx — the single star-rating primitive used everywhere ratings
 * appear: PLP cards, PDP header, the review list, and the write-a-review form.
 *
 * Two modes:
 *  - display (default): read-only stars with optional value + count caption.
 *  - interactive (onChange set): a radiogroup the user sets with mouse or
 *    keyboard (arrows / number keys), used in the submission form.
 */
import { useState } from 'react'
import { Star } from '../components/icons'

type DisplayProps = {
  /** 0–5, may be fractional for averages. */
  value: number
  /** Pixel size of each star. */
  size?: number
  /** When set, renders the numeric average and review count beside the stars. */
  count?: number
  /** Show the numeric value (e.g. "4.3") even without a count. */
  showValue?: boolean
  className?: string
}

export function StarRating({
  value,
  size = 14,
  count,
  showValue,
  className = '',
}: DisplayProps) {
  const rounded = Math.round(value * 2) / 2 // nearest half
  return (
    <span className={`inline-flex items-center gap-1.5 ${className}`}>
      <span
        className="inline-flex items-center gap-0.5"
        role="img"
        aria-label={`Rated ${value.toFixed(1)} out of 5`}
      >
        {[1, 2, 3, 4, 5].map((i) => {
          const fill = rounded >= i ? 1 : rounded >= i - 0.5 ? 0.5 : 0
          return <StarGlyph key={i} fill={fill} size={size} />
        })}
      </span>
      {(showValue || count != null) && (
        <span className="uc-tight text-[0.7rem] text-ink-faint">
          {(showValue || count != null) && (
            <span className="tnum text-ink-soft">{value.toFixed(1)}</span>
          )}
          {count != null && (
            <>
              {' '}
              <span className="tnum">({count})</span>
            </>
          )}
        </span>
      )}
    </span>
  )
}

/** A single star showing empty / half / full fill, tinted with the accent. */
function StarGlyph({ fill, size }: { fill: 0 | 0.5 | 1; size: number }) {
  if (fill === 1) {
    return <Star size={size} strokeWidth={1.5} className="text-accent fill-accent" aria-hidden />
  }
  if (fill === 0) {
    return <Star size={size} strokeWidth={1.5} className="text-rule-strong" aria-hidden />
  }
  // Half: clip a filled star over an empty one.
  return (
    <span className="relative inline-flex" style={{ width: size, height: size }} aria-hidden>
      <Star size={size} strokeWidth={1.5} className="absolute inset-0 text-rule-strong" />
      <span className="absolute inset-0 overflow-hidden" style={{ width: size / 2 }}>
        <Star size={size} strokeWidth={1.5} className="text-accent fill-accent" />
      </span>
    </span>
  )
}

type InputProps = {
  value: number
  onChange: (rating: number) => void
  size?: number
  disabled?: boolean
}

/** Interactive 1–5 picker for the review form. Keyboard accessible. */
export function StarRatingInput({ value, onChange, size = 26, disabled }: InputProps) {
  const [hover, setHover] = useState(0)
  const shown = hover || value

  return (
    <div
      role="radiogroup"
      aria-label="Your rating"
      className="inline-flex items-center gap-1"
      onMouseLeave={() => setHover(0)}
    >
      {[1, 2, 3, 4, 5].map((i) => (
        <button
          key={i}
          type="button"
          role="radio"
          aria-checked={value === i}
          aria-label={`${i} ${i === 1 ? 'star' : 'stars'}`}
          disabled={disabled}
          onMouseEnter={() => setHover(i)}
          onFocus={() => setHover(i)}
          onClick={() => onChange(i)}
          className="cursor-pointer p-0.5 text-ink-faint hover:text-accent transition-colors disabled:cursor-not-allowed disabled:opacity-40"
        >
          <Star
            size={size}
            strokeWidth={1.5}
            className={shown >= i ? 'text-accent fill-accent' : 'text-rule-strong'}
            aria-hidden
          />
        </button>
      ))}
    </div>
  )
}
