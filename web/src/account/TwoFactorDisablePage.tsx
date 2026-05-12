import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { api, ApiError } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'

export function TwoFactorDisablePage() {
  const { isAuthed } = useAuth()
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isAuthed) return <Navigate to="/login" replace />

  async function handleDisable(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.disable2FA(code)
      navigate('/account')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Disable failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-md">
      <Masthead
        number="02"
        eyebrow="Two-factor"
        title="Disable two-factor."
        caption="Enter a current TOTP code or one of your recovery codes to confirm."
      />

      <form onSubmit={handleDisable} className="space-y-8">
        <Field
          label="Verification code"
          inputMode="numeric"
          autoComplete="one-time-code"
          required
          value={code}
          onChange={(e) => setCode(e.target.value)}
          hint="6-digit TOTP code or a recovery code."
        />

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6">
          <Button type="submit" disabled={busy}>
            {busy ? 'Disabling.' : 'Disable two-factor'}
          </Button>
          <Link to="/account" className="text-sm text-ink-soft hover:text-ink">
            Cancel.
          </Link>
        </div>
      </form>
    </Page>
  )
}
