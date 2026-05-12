import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { request, ApiError } from '../lib/api'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'

type RefreshResponse = {
  access_token: string
  refresh_token: string
  expires_at: string
  token_type: string
}

export function TokenTesterPage() {
  const { isAuthed } = useAuth()
  const [email, setEmail] = useState('customer@shop.com')
  const [password, setPassword] = useState('customer123')
  const [loginResult, setLoginResult] = useState<RefreshResponse | null>(null)
  const [currentRefresh, setCurrentRefresh] = useState('')
  const [replayInput, setReplayInput] = useState('')
  const [log, setLog] = useState<string[]>([])
  const [busy, setBusy] = useState(false)

  if (!isAuthed) return <Navigate to="/login" replace />

  function append(line: string) {
    setLog((prev) => [`${new Date().toISOString().slice(11, 19)}  ${line}`, ...prev])
  }

  async function doLogin() {
    setBusy(true)
    try {
      const res = await request<RefreshResponse>('/api/v1/auth/login', {
        method: 'POST',
        body: { email, password },
      })
      setLoginResult(res)
      setCurrentRefresh(res.refresh_token)
      append(`login ok — refresh ${res.refresh_token.slice(0, 12)}…`)
    } catch (err) {
      append(`login failed — ${err instanceof ApiError ? err.message : err}`)
    } finally {
      setBusy(false)
    }
  }

  async function doRefresh() {
    if (!currentRefresh) {
      append('no refresh token to rotate.')
      return
    }
    setBusy(true)
    const stale = currentRefresh
    try {
      const res = await request<RefreshResponse>('/api/v1/auth/refresh', {
        method: 'POST',
        body: { refresh_token: currentRefresh },
      })
      setCurrentRefresh(res.refresh_token)
      append(
        `rotated — old ${stale.slice(0, 12)}… → new ${res.refresh_token.slice(0, 12)}…`,
      )
    } catch (err) {
      append(`rotate failed — ${err instanceof ApiError ? err.message : err}`)
    } finally {
      setBusy(false)
    }
  }

  async function doReplay() {
    const token = replayInput || ''
    if (!token) {
      append('paste a previous refresh token first.')
      return
    }
    setBusy(true)
    try {
      const res = await request<RefreshResponse>('/api/v1/auth/refresh', {
        method: 'POST',
        body: { refresh_token: token },
      })
      append(`unexpected success — got ${res.refresh_token.slice(0, 12)}…`)
    } catch (err) {
      append(
        `replay rejected as expected — ${err instanceof ApiError ? err.message : err}`,
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-4xl">
      <Masthead
        number="01"
        eyebrow="Developer"
        title="Token rotation tester."
        caption="Demonstrates single-use refresh tokens and replay detection. Log in, rotate, then paste an old refresh token to confirm the server invalidates the family."
      />

      <section className="grid grid-cols-1 lg:grid-cols-[1fr_1fr] gap-x-12 gap-y-8 mb-12">
        <div className="space-y-6">
          <div className="uc-tight text-[0.7rem] text-ink-faint">
            <span className="tnum">01</span> · Login
          </div>
          <Field
            label="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <Field
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Button onClick={doLogin} disabled={busy}>
            Log in
          </Button>
        </div>

        <div className="space-y-6">
          <div className="uc-tight text-[0.7rem] text-ink-faint">
            <span className="tnum">02</span> · Tokens
          </div>
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-1">
              Access token (in-memory)
            </p>
            <code className="block bg-raised border border-rule px-3 py-2 text-xs font-mono break-all text-ink-soft min-h-[2.5rem]">
              {loginResult?.access_token ?? '—'}
            </code>
          </div>
          <div>
            <p className="uc-tight text-[0.7rem] text-ink-faint mb-1">
              Current refresh token
            </p>
            <code className="block bg-raised border border-rule px-3 py-2 text-xs font-mono break-all text-ink-soft min-h-[2.5rem]">
              {currentRefresh || '—'}
            </code>
          </div>
          <Button variant="ghost" onClick={doRefresh} disabled={busy || !currentRefresh}>
            Rotate refresh
          </Button>
        </div>
      </section>

      <section className="border-t border-rule pt-8 mb-12">
        <div className="uc-tight text-[0.7rem] text-ink-faint mb-4">
          <span className="tnum">03</span> · Replay
        </div>
        <p className="text-sm text-ink-soft max-w-[55ch] mb-4">
          Paste any old refresh token (eg copy the value before clicking{' '}
          <em>Rotate</em>) and send it. The server rejects it and revokes the
          rest of the family.
        </p>
        <Field
          label="Old refresh token"
          value={replayInput}
          onChange={(e) => setReplayInput(e.target.value)}
        />
        <div className="mt-4">
          <Button onClick={doReplay} disabled={busy}>
            Send replay
          </Button>
        </div>
      </section>

      <section className="border-t border-rule pt-8">
        <div className="uc-tight text-[0.7rem] text-ink-faint mb-4">
          <span className="tnum">04</span> · Log
        </div>
        <pre className="bg-raised border border-rule px-3 py-3 text-xs font-mono leading-relaxed text-ink-soft min-h-[10rem] max-h-[24rem] overflow-auto whitespace-pre-wrap">
          {log.length === 0 ? 'No activity yet.' : log.join('\n')}
        </pre>
      </section>
    </Page>
  )
}
