import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { api, ApiError, type ModerationReview } from '../lib/api'
import { StarRating } from '../reviews/StarRating'

const FILTERS = [
  { value: 'pending', label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'hidden', label: 'Hidden' },
  { value: '', label: 'All' },
]

export function AdminReviewsPage() {
  const [reviews, setReviews] = useState<ModerationReview[]>([])
  const [filter, setFilter] = useState('pending')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const refresh = useCallback(() => {
    setLoading(true)
    setError(null)
    api
      .adminListReviews(filter || undefined)
      .then((res) => setReviews(res.reviews))
      .catch((e) => setError(e instanceof ApiError ? e.message : 'Could not load reviews.'))
      .finally(() => setLoading(false))
  }, [filter])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh()
  }, [refresh])

  async function setStatus(id: string, status: 'approved' | 'hidden' | 'pending') {
    setBusyId(id)
    try {
      await api.adminSetReviewStatus(id, status)
      refresh()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not update review.')
    } finally {
      setBusyId(null)
    }
  }

  async function remove(id: string) {
    setBusyId(id)
    try {
      await api.adminDeleteReview(id)
      refresh()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not delete review.')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Moderation"
        title="Reviews."
        caption="Approve, hide, or remove customer reviews. Only approved reviews appear on the storefront."
      />

      <div className="flex flex-wrap items-center gap-2 mb-8">
        {FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => setFilter(f.value)}
            className={`text-xs px-3 py-1.5 border transition-colors cursor-pointer ${
              filter === f.value
                ? 'border-accent text-accent'
                : 'border-rule text-ink-soft hover:border-ink hover:text-ink'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}
      {loading ? (
        <p className="text-sm text-ink-soft">Loading.</p>
      ) : reviews.length === 0 ? (
        <p className="text-sm text-ink-faint">No reviews in this state.</p>
      ) : (
        <ul className="divide-y divide-rule">
          {reviews.map((r) => (
            <li key={r.id} className="py-6 flex flex-col gap-3">
              <div className="flex items-center justify-between gap-4">
                <StarRating value={r.rating} size={14} />
                <span className="uc-tight text-[0.7rem] text-ink-faint">
                  {r.status}
                  <span aria-hidden className="text-rule-strong mx-2">/</span>
                  {new Date(r.created_at).toLocaleDateString()}
                </span>
              </div>
              <Link
                to={`/product/${r.product_id}`}
                className="text-xs text-ink-soft hover:text-accent underline underline-offset-4 decoration-rule-strong self-start"
              >
                {r.product_name}
              </Link>
              {r.comment && <p className="text-ink-soft leading-relaxed">{r.comment}</p>}
              <div className="flex flex-wrap gap-2">
                {r.status !== 'approved' && (
                  <ActionBtn disabled={busyId === r.id} onClick={() => setStatus(r.id, 'approved')}>
                    Approve
                  </ActionBtn>
                )}
                {r.status !== 'hidden' && (
                  <ActionBtn disabled={busyId === r.id} onClick={() => setStatus(r.id, 'hidden')}>
                    Hide
                  </ActionBtn>
                )}
                <button
                  type="button"
                  disabled={busyId === r.id}
                  onClick={() => remove(r.id)}
                  className="text-xs px-3 py-1.5 border border-negative text-negative hover:bg-negative hover:text-on-accent transition-colors cursor-pointer disabled:opacity-40"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Page>
  )
}

function ActionBtn({
  children,
  disabled,
  onClick,
}: {
  children: React.ReactNode
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="text-xs px-3 py-1.5 border border-rule text-ink-soft hover:border-accent hover:text-accent transition-colors cursor-pointer disabled:opacity-40"
    >
      {children}
    </button>
  )
}
