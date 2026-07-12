/* ProfileSection.tsx — name (editable), email (read-only), member since.
 * Saving refreshes the auth context so the nav shows the new name at once.
 */
import { useEffect, useState } from 'react'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../auth/useAuth'
import { Field } from '../components/Field'
import { Button } from '../components/Button'

export function ProfileSection() {
  const { initializing, refreshMe } = useAuth()
  const [nameInput, setNameInput] = useState('')
  const [email, setEmail] = useState('')
  const [memberSince, setMemberSince] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (initializing) return
    let cancelled = false
    api
      .me()
      .then((me) => {
        if (cancelled) return
        setNameInput(me.name)
        setEmail(me.email)
        setMemberSince(me.created_at)
        setLoaded(true)
      })
      .catch(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [initializing])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    setSaved(false)
    try {
      const me = await api.updateProfile(nameInput.trim())
      setNameInput(me.name)
      setSaved(true)
      await refreshMe()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save your profile.')
    } finally {
      setBusy(false)
    }
  }

  if (!loaded) return null

  return (
    <form onSubmit={handleSubmit} className="space-y-5 max-w-md">
      <Field
        label="Name"
        type="text"
        maxLength={100}
        autoComplete="name"
        value={nameInput}
        onChange={(e) => setNameInput(e.target.value)}
      />
      <div>
        <p className="uc-tight text-[0.7rem] text-ink-faint">Email</p>
        <p className="text-sm text-ink mt-1.5">{email}</p>
      </div>
      {memberSince && (
        <div>
          <p className="uc-tight text-[0.7rem] text-ink-faint">Member since</p>
          <p className="text-sm text-ink mt-1.5">
            {new Date(memberSince).toLocaleDateString(undefined, {
              year: 'numeric',
              month: 'long',
            })}
          </p>
        </div>
      )}

      {error && (
        <p className="text-sm text-accent" role="alert">
          {error}
        </p>
      )}
      {saved && (
        <p className="text-sm text-positive" role="status">
          Profile saved.
        </p>
      )}

      <Button type="submit" disabled={busy}>
        {busy ? 'Saving…' : 'Save profile'}
      </Button>
    </form>
  )
}
