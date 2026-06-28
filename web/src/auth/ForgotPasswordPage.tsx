import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError } from '../lib/api'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [busy, setBusy] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.forgotPassword(email)
      setSent(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Request failed.')
    } finally {
      setBusy(false)
    }
  }

  if (sent) {
    return (
      <Page width="max-w-md">
        <Masthead
          eyebrow="Recover"
          title="Check your inbox."
          caption={
            <>
              If an account exists for <span className="text-ink">{email}</span>,
              a reset link is on its way. The token expires in one hour.
            </>
          }
        />
        <Link to="/login" className="text-sm text-ink-soft hover:text-ink">
          Back to log in.
        </Link>
      </Page>
    )
  }

  return (
    <Page width="max-w-md">
      <Masthead
        eyebrow="Recover"
        title="Reset password."
        caption="Enter the email tied to your account. If we find it, we'll send a one-time link."
      />

      <form onSubmit={handleSubmit} className="space-y-8">
        <Field
          label="Email"
          type="email"
          required
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6">
          <Button type="submit" disabled={busy}>
            {busy ? 'Sending.' : 'Send reset link'}
          </Button>
          <Link to="/login" className="text-sm text-ink-soft hover:text-ink">
            Back to log in.
          </Link>
        </div>
      </form>
    </Page>
  )
}
