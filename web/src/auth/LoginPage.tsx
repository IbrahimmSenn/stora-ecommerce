import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from './useAuth'
import { useCart } from '../cart/useCart'
import { MergePromptModal } from './MergePromptModal'
import { ApiError, api } from '../lib/api'
import type { Cart } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'

export function LoginPage() {
  const { login } = useAuth()
  const { refresh, fetchMergeStatus } = useCart()
  const navigate = useNavigate()
  const [params] = useSearchParams()

  const initialEmail = params.get('email') ?? ''
  const justReset = params.get('reset') === '1'

  const [email, setEmail] = useState(initialEmail)
  const [password, setPassword] = useState('')
  const [totp, setTotp] = useState('')
  const [twoFactorNeeded, setTwoFactorNeeded] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState<{ guest: Cart; user: Cart } | null>(
    null,
  )

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(email, password, twoFactorNeeded ? totp : undefined)
      const status = await fetchMergeStatus()
      if (status.conflict && status.guest_cart && status.user_cart) {
        setConflict({ guest: status.guest_cart, user: status.user_cart })
        return
      }
      await refresh()
      navigate('/')
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 403 && err.message.includes('2fa')) {
          setTwoFactorNeeded(true)
          setError('Enter the 6-digit code from your authenticator.')
        } else if (err.status === 401 && err.message.includes('2fa')) {
          setError('Invalid 2FA code.')
        } else {
          setError(err.message)
        }
      } else {
        setError('Login failed.')
      }
    } finally {
      setBusy(false)
    }
  }

  function handleOAuth(provider: 'google' | 'facebook') {
    window.location.assign(api.oauthRedirectUrl(provider))
  }

  async function handleResolved() {
    setConflict(null)
    await refresh()
    navigate('/')
  }

  return (
    <Page width="max-w-md">
      <Masthead
        number="01"
        eyebrow="Account"
        title="Log in."
        caption={
          justReset
            ? 'Password updated. Log in with your new password.'
            : undefined
        }
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
        <Field
          label="Password"
          type="password"
          required
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        {twoFactorNeeded && (
          <Field
            label="Two-factor code"
            inputMode="numeric"
            pattern="[0-9]*"
            autoComplete="one-time-code"
            placeholder="123456"
            value={totp}
            onChange={(e) => setTotp(e.target.value)}
            hint="6-digit code from your authenticator, or one of your recovery codes."
          />
        )}

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6 pt-2">
          <Button type="submit" disabled={busy}>
            {busy ? 'Signing in.' : 'Log in'}
          </Button>
          <Link
            to="/forgot-password"
            className="text-sm text-ink-soft hover:text-ink"
          >
            Forgot password?
          </Link>
        </div>
      </form>

      <div className="mt-16 pt-8 border-t border-rule">
        <p className="uc-tight text-[0.7rem] text-ink-faint mb-4">Or via</p>
        <div className="flex gap-3">
          <Button
            variant="ghost"
            type="button"
            onClick={() => handleOAuth('google')}
          >
            Google
          </Button>
          <Button
            variant="ghost"
            type="button"
            onClick={() => handleOAuth('facebook')}
          >
            Facebook
          </Button>
        </div>
        <p className="text-xs text-ink-faint mt-3">
          Requires OAuth credentials configured in the server's .env.
        </p>
      </div>

      <p className="mt-12 text-sm text-ink-soft">
        New here?{' '}
        <Link to="/register" className="text-ink underline underline-offset-4">
          Create an account.
        </Link>
      </p>

      {conflict && (
        <MergePromptModal
          guestCart={conflict.guest}
          userCart={conflict.user}
          onResolved={handleResolved}
        />
      )}
    </Page>
  )
}
