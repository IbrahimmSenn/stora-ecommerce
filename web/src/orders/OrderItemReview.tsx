/* OrderItemReview.tsx — a single inline review widget shown for one purchased
 * item on the order confirmation page. Lets a buyer rate and review a product
 * without leaving the page. Reviews auto-approve, so a submission is live
 * immediately. Reuses the shared StarRatingInput and the reviews API.
 */
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../lib/api'
import type { OrderItem, ReviewEligibility } from '../lib/api'
import { StarRatingInput } from '../reviews/StarRating'

export function OrderItemReview({ item }: { item: OrderItem }) {
  const productId = item.product_id ?? ''
  const [eligibility, setEligibility] = useState<ReviewEligibility | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  useEffect(() => {
    if (!productId) return
    let cancelled = false
    api
      .reviewEligibility(productId)
      .then((e) => {
        if (!cancelled) setEligibility(e)
      })
      .catch(() => {
        // Treat an eligibility failure as "can't review here" — stay quiet.
      })
      .finally(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [productId])

  if (!productId || !loaded) return null

  // Nothing to offer: not a verified buyer for this item, and no prior review.
  if (!done && !eligibility?.already_reviewed && !eligibility?.can_review) return null

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (rating < 1) {
      setError('Please choose a star rating.')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await api.createReview(productId, rating, comment.trim())
      setDone(true)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'already_reviewed') {
        setDone(true)
      } else if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Could not submit your review.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <li className="flex gap-4 py-5">
      <Link to={`/product/${productId}`} className="shrink-0" title={item.product_name}>
        <span className="block h-14 w-14 overflow-hidden bg-sunken" style={{ borderRadius: 0 }}>
          {item.thumbnail_url ? (
            <img
              src={item.thumbnail_url}
              alt={item.product_name}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          ) : (
            <span aria-hidden className="block h-full w-full" />
          )}
        </span>
      </Link>

      <div className="flex-1 min-w-0">
        <Link
          to={`/product/${productId}`}
          className="text-ink hover:text-accent transition-colors underline-offset-4 hover:underline decoration-rule-strong"
        >
          {item.product_name}
        </Link>

        {done || eligibility?.already_reviewed ? (
          <p className="text-sm text-ink-soft mt-2" role="status">
            {done ? 'Thanks — your review is live.' : 'You’ve already reviewed this.'}
          </p>
        ) : (
          <form onSubmit={submit} className="mt-3 flex flex-col gap-3">
            <StarRatingInput value={rating} onChange={setRating} disabled={submitting} size={22} />
            <textarea
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              maxLength={2000}
              rows={2}
              placeholder="Share what stood out (optional)."
              className="bg-transparent border border-rule px-3 py-2 text-sm text-ink placeholder:text-ink-faint focus:border-accent outline-none resize-y"
            />
            {error && <p className="text-xs text-accent" role="alert">{error}</p>}
            <button
              type="submit"
              disabled={submitting}
              className="bg-accent hover:bg-accent-soft text-on-accent transition-colors px-5 py-2 text-sm self-start disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            >
              {submitting ? 'Submitting.' : 'Submit review'}
            </button>
          </form>
        )}
      </div>
    </li>
  )
}
