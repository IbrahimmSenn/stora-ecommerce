import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError } from '../lib/api'
import { PasswordChecklist } from './PasswordChecklist'
import { passwordIsStrong } from './passwordCriteria'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    if (!passwordIsStrong(password)) {
      setError(
        'Password must be at least 8 characters and include an uppercase letter, a lowercase letter, a number, and a symbol.',
      )
      return
    }
    if (password !== confirm) {
      setError('Passwords do not match.')
      return
    }
    if (!token) {
      setError('Reset token missing from URL.')
      return
    }
    setBusy(true)
    try {
      await api.resetPassword(token, password)
      navigate('/login?reset=1')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Reset failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-md">
      <Masthead
        eyebrow="Recover"
        title="Set a new password"
        caption={
          token
            ? 'Choose a new password for your account below.'
            : 'No reset token found in the link. Open the reset link from your email.'
        }
      />

      <form onSubmit={handleSubmit} className="space-y-8">
        <Field
          label="New password"
          type="password"
          required
          autoComplete="new-password"
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {password.length > 0 && <PasswordChecklist password={password} />}
        <Field
          label="Confirm new password"
          type="password"
          required
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6">
          <Button type="submit" disabled={busy || !token}>
            {busy ? 'Saving…' : 'Save new password'}
          </Button>
          <Link to="/login" className="text-sm text-ink-soft hover:text-ink">
            Back to log in
          </Link>
        </div>
      </form>
    </Page>
  )
}
