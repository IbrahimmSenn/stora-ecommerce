/* PasswordSection.tsx — change password while signed in. Other sessions lose
 * their refresh tokens on success, matching the reset flow.
 */
import { useState } from 'react'
import { api, ApiError } from '../lib/api'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { PasswordChecklist } from '../auth/PasswordChecklist'
import { passwordIsStrong } from '../auth/passwordCriteria'

export function PasswordSection() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setDone(false)
    if (!passwordIsStrong(next)) {
      setError(
        'New password must be at least 8 characters and include an uppercase letter, a lowercase letter, a number, and a symbol.',
      )
      return
    }
    if (next !== confirm) {
      setError('New passwords do not match.')
      return
    }
    setBusy(true)
    try {
      await api.changePassword(current, next)
      setCurrent('')
      setNext('')
      setConfirm('')
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not change your password.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5 max-w-md">
      <Field
        label="Current password"
        type="password"
        required
        autoComplete="current-password"
        value={current}
        onChange={(e) => setCurrent(e.target.value)}
      />
      <Field
        label="New password"
        type="password"
        required
        autoComplete="new-password"
        minLength={8}
        value={next}
        onChange={(e) => setNext(e.target.value)}
      />
      {next.length > 0 && <PasswordChecklist password={next} />}
      <Field
        label="Confirm new password"
        type="password"
        required
        autoComplete="new-password"
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
      />

      {error && (
        <p className="text-sm text-accent" role="alert">
          {error}
        </p>
      )}
      {done && (
        <p className="text-sm text-positive" role="status">
          Password updated. Other devices will be signed out.
        </p>
      )}

      <Button type="submit" disabled={busy}>
        {busy ? 'Updating…' : 'Change password'}
      </Button>
    </form>
  )
}
