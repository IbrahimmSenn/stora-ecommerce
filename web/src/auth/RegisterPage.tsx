import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { Seo } from '../components/Seo'
import { api, ApiError } from '../lib/api'
import { captchaEnabled, getCaptchaToken } from '../lib/captcha'
import { PasswordChecklist } from './PasswordChecklist'
import { passwordIsStrong } from './passwordCriteria'

export function RegisterPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [captchaActive, setCaptchaActive] = useState(false)

  useEffect(() => {
    let cancelled = false
    captchaEnabled().then((on) => {
      if (!cancelled) setCaptchaActive(on)
    })
    return () => {
      cancelled = true
    }
  }, [])

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
      <Seo title="Create an account" description="Create a Stora account to save addresses, track orders, and check out faster — secure sign-up with optional two-factor." />
      <Masthead eyebrow="Account" title="Create account" />

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
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {password.length > 0 && <PasswordChecklist password={password} />}
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
            {busy ? 'Creating account…' : 'Create account'}
          </Button>
          <Link to="/login" className="text-sm text-ink-soft hover:text-ink">
            Already have an account?
          </Link>
        </div>

        {captchaActive && (
          <p className="text-xs text-ink-faint pt-6">Protected by reCAPTCHA.</p>
        )}
      </form>
    </Page>
  )
}
