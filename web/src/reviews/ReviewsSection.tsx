/* ReviewsSection.tsx — the reviews block on the product detail page.
 *
 * Shows the rating summary (average + per-star distribution), a sortable list
 * of approved reviews with helpful voting, and — for signed-in shoppers who
 * bought the product — a write-a-review form. New reviews are submitted as
 * pending and surface only after admin moderation, which the form makes clear.
 */
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../lib/api'
import type {
  PublicReview,
  ReviewEligibility,
  ReviewListResult,
  ReviewSort,
} from '../lib/api'
import { useAuth } from '../auth/useAuth'
import { useToast } from '../components/useToast'
import { ThumbsUp } from '../components/icons'
import { StarRating, StarRatingInput } from './StarRating'

const SORTS: { value: ReviewSort; label: string }[] = [
  { value: 'helpful', label: 'Most helpful' },
  { value: 'newest', label: 'Newest' },
  { value: 'highest', label: 'Highest rated' },
  { value: 'lowest', label: 'Lowest rated' },
]

export function ReviewsSection({ productId }: { productId: string }) {
  const { isAuthed, initializing } = useAuth()
  const [data, setData] = useState<ReviewListResult | null>(null)
  const [sort, setSort] = useState<ReviewSort>('helpful')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [eligibility, setEligibility] = useState<ReviewEligibility | null>(null)

  const load = useCallback(
    (s: ReviewSort) => {
      setLoading(true)
      api
        .listReviews(productId, s)
        .then(setData)
        .catch((e) => setError(e instanceof Error ? e.message : 'Could not load reviews.'))
        .finally(() => setLoading(false))
    },
    [productId],
  )

  useEffect(() => {
    // Reacting to external inputs (product id / chosen sort); the synchronous
    // setLoading inside load is the right shape here.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load(sort)
  }, [load, sort])

  useEffect(() => {
    if (initializing || !isAuthed) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setEligibility(null)
      return
    }
    api
      .reviewEligibility(productId)
      .then(setEligibility)
      .catch(() => setEligibility(null))
  }, [productId, isAuthed, initializing])

  const total = data?.total ?? 0

  return (
    <section aria-labelledby="reviews-heading" className="border-t border-rule pt-12 mt-16">
      <header className="flex flex-col gap-1 mb-10">
        <span className="uc-tight text-[0.7rem] text-ink-faint">
          Reviews
        </span>
        <h2 id="reviews-heading" className="font-display text-[clamp(1.5rem,3vw,2rem)] leading-tight text-ink font-bold">
          What buyers say.
        </h2>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-x-16 gap-y-12">
        {/* Summary + form */}
        <div className="lg:col-span-4 flex flex-col gap-8">
          <RatingSummary data={data} loading={loading} />
          <ReviewForm
            productId={productId}
            isAuthed={isAuthed}
            initializing={initializing}
            eligibility={eligibility}
            onSubmitted={() => {
              api.reviewEligibility(productId).then(setEligibility).catch(() => {})
            }}
          />
        </div>

        {/* List */}
        <div className="lg:col-span-8">
          {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}

          {total > 0 && (
            <div className="flex items-center justify-between gap-4 mb-8">
              <p className="uc-tight text-[0.7rem] text-ink-faint">
                <span className="tnum">{total}</span> {total === 1 ? 'review' : 'reviews'}
              </p>
              <label className="flex items-center gap-2 text-xs text-ink-soft">
                <span className="uc-tight text-[0.7rem] text-ink-faint">Sort</span>
                <select
                  value={sort}
                  onChange={(e) => setSort(e.target.value as ReviewSort)}
                  className="bg-transparent border border-rule px-2 py-1 text-ink cursor-pointer focus:border-accent outline-none"
                >
                  {SORTS.map((s) => (
                    <option key={s.value} value={s.value}>{s.label}</option>
                  ))}
                </select>
              </label>
            </div>
          )}

          {loading && !data ? (
            <p className="text-sm text-ink-soft">Loading.</p>
          ) : total === 0 ? (
            <p className="text-sm text-ink-soft">
              No reviews yet. Bought this? Be the first to review it.
            </p>
          ) : (
            <ul className="flex flex-col">
              {data!.reviews.map((r) => (
                <ReviewItem key={r.id} review={r} canVote={isAuthed} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </section>
  )
}

function RatingSummary({ data, loading }: { data: ReviewListResult | null; loading: boolean }) {
  if (loading && !data) return null
  const total = data?.total ?? 0
  if (total === 0) {
    return (
      <div className="flex flex-col gap-1">
        <p className="font-display text-4xl text-ink font-bold tnum">—</p>
        <p className="text-xs text-ink-faint">No ratings yet.</p>
      </div>
    )
  }
  const dist = data!.distribution
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-end gap-3">
        <p className="font-display text-5xl text-ink font-bold tnum leading-none">
          {data!.avg_rating.toFixed(1)}
        </p>
        <div className="flex flex-col gap-1 pb-1">
          <StarRating value={data!.avg_rating} size={16} />
          <p className="text-xs text-ink-faint">
            <span className="tnum">{total}</span> {total === 1 ? 'review' : 'reviews'}
          </p>
        </div>
      </div>
      <ul className="flex flex-col gap-1.5">
        {[5, 4, 3, 2, 1].map((star) => {
          const n = dist[String(star)] ?? 0
          const pct = total > 0 ? (n / total) * 100 : 0
          return (
            <li key={star} className="flex items-center gap-3 text-xs">
              <span className="uc-tight text-[0.7rem] text-ink-faint tnum w-3">{star}</span>
              <span className="flex-1 h-1.5 bg-sunken overflow-hidden">
                <span className="block h-full bg-accent" style={{ width: `${pct}%` }} />
              </span>
              <span className="tnum text-ink-faint w-6 text-right">{n}</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

function ReviewItem({ review, canVote }: { review: PublicReview; canVote: boolean }) {
  const [voted, setVoted] = useState(review.voted_by_me)
  const [count, setCount] = useState(review.helpful_count)
  const [busy, setBusy] = useState(false)

  async function toggle() {
    if (busy) return
    setBusy(true)
    const next = !voted
    // optimistic
    setVoted(next)
    setCount((c) => c + (next ? 1 : -1))
    try {
      await api.voteHelpful(review.id, next)
    } catch {
      setVoted(!next)
      setCount((c) => c + (next ? -1 : 1))
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="border-t border-rule py-6 first:border-t-0 first:pt-0 flex flex-col gap-3">
      <div className="flex items-center justify-between gap-4">
        <StarRating value={review.rating} size={14} />
        <span className="uc-tight text-[0.7rem] text-ink-faint">
          {review.mine_to_edit ? 'Your review' : 'Verified buyer'}
          <span aria-hidden className="text-rule-strong mx-2">/</span>
          {new Date(review.created_at).toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
          })}
        </span>
      </div>
      {review.comment && (
        <p className="text-ink-soft leading-relaxed whitespace-pre-line">{review.comment}</p>
      )}
      <div>
        <button
          type="button"
          onClick={toggle}
          disabled={!canVote || busy || review.mine_to_edit}
          title={
            !canVote
              ? 'Sign in to vote'
              : review.mine_to_edit
                ? "You can't vote on your own review"
                : 'Mark helpful'
          }
          aria-pressed={voted}
          className={`inline-flex items-center gap-2 text-xs border px-3 py-1.5 transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 ${
            voted
              ? 'border-accent text-accent'
              : 'border-rule text-ink-soft hover:border-accent hover:text-accent'
          }`}
        >
          <ThumbsUp size={13} strokeWidth={1.5} aria-hidden />
          Helpful
          {count > 0 && <span className="tnum">({count})</span>}
        </button>
      </div>
    </li>
  )
}

function ReviewForm({
  productId,
  isAuthed,
  initializing,
  eligibility,
  onSubmitted,
}: {
  productId: string
  isAuthed: boolean
  initializing: boolean
  eligibility: ReviewEligibility | null
  onSubmitted: () => void
}) {
  const { show: showToast } = useToast()
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  if (initializing) return null

  if (!isAuthed) {
    return (
      <div className="border-t border-rule pt-6">
        <p className="text-sm text-ink-soft">
          <Link to="/login" className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors">
            Sign in
          </Link>{' '}
          to review a product you've bought.
        </p>
      </div>
    )
  }

  if (done) {
    return (
      <div className="border-t border-rule pt-6">
        <p className="text-sm text-ink-soft" role="status">
          Thanks — your review was submitted and is awaiting moderation.
        </p>
      </div>
    )
  }

  if (eligibility?.already_reviewed) {
    return (
      <div className="border-t border-rule pt-6">
        <p className="text-sm text-ink-soft">
          You've reviewed this product
          {eligibility.existing_pending ? " — it's awaiting moderation." : '.'}
        </p>
      </div>
    )
  }

  if (eligibility && !eligibility.has_purchased) {
    return (
      <div className="border-t border-rule pt-6">
        <p className="text-sm text-ink-soft">
          Only verified buyers can review this product.
        </p>
      </div>
    )
  }

  if (!eligibility?.can_review) return null

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
      showToast('Review submitted for moderation.')
      onSubmitted()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Could not submit your review.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={submit} className="border-t border-rule pt-6 flex flex-col gap-4">
      <h3 className="uc-tight text-[0.7rem] text-ink-faint">Write a review</h3>
      <StarRatingInput value={rating} onChange={setRating} disabled={submitting} />
      <textarea
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        maxLength={2000}
        rows={4}
        placeholder="Share what stood out (optional)."
        className="bg-transparent border border-rule px-3 py-2 text-sm text-ink placeholder:text-ink-faint focus:border-accent outline-none resize-y"
      />
      {error && <p className="text-xs text-accent" role="alert">{error}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="bg-accent hover:bg-accent-soft text-on-accent transition-colors px-5 py-2.5 text-sm self-start disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
      >
        {submitting ? 'Submitting.' : 'Submit review'}
      </button>
    </form>
  )
}
