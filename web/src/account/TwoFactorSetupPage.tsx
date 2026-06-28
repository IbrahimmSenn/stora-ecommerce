import { useEffect, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { api, ApiError } from '../lib/api'
import type { TwoFactorSetupResponse } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'

export function TwoFactorSetupPage() {
  const { isAuthed } = useAuth()
  const navigate = useNavigate()
  const [setup, setSetup] = useState<TwoFactorSetupResponse | null>(null)
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  useEffect(() => {
    if (!isAuthed) return
    let cancelled = false
    api
      .setup2FA()
      .then((res) => {
        if (!cancelled) setSetup(res)
      })
      .catch((e) => {
        if (cancelled) return
        setError(e instanceof ApiError ? e.message : 'Setup failed.')
      })
    return () => {
      cancelled = true
    }
  }, [isAuthed])

  if (!isAuthed) return <Navigate to="/login" replace />

  async function handleEnable(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.enable2FA(code)
      setDone(true)
      setTimeout(() => navigate('/account'), 1200)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Activation failed.')
    } finally {
      setBusy(false)
    }
  }

  if (done) {
    return (
      <Page width="max-w-md">
        <Masthead
          eyebrow="Two-factor"
          title="Activated."
          caption="From now on, log-in requires a TOTP code."
        />
      </Page>
    )
  }

  if (!setup) {
    return (
      <Page width="max-w-md">
        <Masthead eyebrow="Two-factor" title="Setting up." />
        {error ? (
          <p className="text-sm text-accent">{error}</p>
        ) : (
          <p className="text-sm text-ink-soft">Generating QR code.</p>
        )}
      </Page>
    )
  }

  return (
    <Page width="max-w-2xl">
      <Masthead
        eyebrow="Two-factor"
        title="Scan the code."
        caption="Open your authenticator app, scan the QR, then enter the 6-digit code below to activate."
      />

      <div className="grid grid-cols-1 md:grid-cols-[14rem_1fr] gap-8 mb-12">
        <div className="bg-raised border border-rule p-3 inline-block self-start">
          <img
            src={`data:image/png;base64,${setup.qr_code}`}
            alt="Authenticator QR code"
            width={196}
            height={196}
            className="block"
          />
        </div>
        <div className="space-y-4">
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">
              Manual entry secret
            </p>
            <code className="block bg-raised px-3 py-2 font-mono text-sm tnum border border-rule break-all">
              {setup.secret}
            </code>
          </div>
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-2">
              Recovery codes — store these somewhere safe
            </p>
            <ol className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm tnum font-mono text-ink">
              {setup.recovery_codes.map((c) => (
                <li key={c}>{c}</li>
              ))}
            </ol>
          </div>
        </div>
      </div>

      <form
        onSubmit={handleEnable}
        className="space-y-6 border-t border-rule pt-8"
      >
        <Field
          label="Verification code"
          inputMode="numeric"
          pattern="[0-9]*"
          maxLength={6}
          required
          autoComplete="one-time-code"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          hint="6 digits from your authenticator."
        />

        {error && <p className="text-sm text-accent">{error}</p>}

        <div className="flex items-center gap-6">
          <Button type="submit" disabled={busy || code.length !== 6}>
            {busy ? 'Activating.' : 'Activate two-factor'}
          </Button>
          <Link to="/account" className="text-sm text-ink-soft hover:text-ink">
            Cancel.
          </Link>
        </div>
      </form>
    </Page>
  )
}
