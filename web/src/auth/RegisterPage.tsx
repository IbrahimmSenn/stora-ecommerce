import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError } from '../lib/api'
import { getCaptchaToken } from '../lib/captcha'

export function RegisterPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    if (password.length < 8) {
      setError('Password must be at least 8 characters.')
      return
    }
    if (password !== confirm) {
      setError('Passwords do not match.')
      return
    }
    setBusy(true)
    try {
      const token = await getCaptchaToken('register')
      await api.register(email, password, token)
      navigate(`/login?email=${encodeURIComponent(email)}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Registration failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-md">
      <Masthead number="01" eyebrow="Account" title="Register." />

      <form onSubmit={handleSubmit} className="space-y-8">
        <Field
          label="Email"
          type="email"
          required
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Field
          label="Password"
          type="password"
          required
          autoComplete="new-password"
          minLength={8}
          hint="At least 8 characters."
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Field
          label="Confirm password"
          type="password"
          required
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6 pt-2">
          <Button type="submit" disabled={busy}>
            {busy ? 'Registering.' : 'Register'}
          </Button>
          <Link to="/login" className="text-sm text-ink-soft hover:text-ink">
            Already have an account?
          </Link>
        </div>

        <p className="text-xs text-ink-faint pt-6">
          Protected by reCAPTCHA when a site key is configured. In development
          with <span className="tnum">SKIP_CAPTCHA=true</span>, the token is
          omitted.
        </p>
      </form>
    </Page>
  )
}
