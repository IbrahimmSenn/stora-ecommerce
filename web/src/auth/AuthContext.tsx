import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { api, ApiError, setAccessToken } from '../lib/api'
import { AuthCtx } from './authCtx'
import type { AuthState } from './authCtx'

type JwtClaims = {
  user_id?: string
  email?: string
  role?: string
}

function decodeJwt(token: string): JwtClaims | null {
  try {
    const payload = token.split('.')[1]
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json) as JwtClaims
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [email, setEmail] = useState<string | null>(null)
  const [role, setRole] = useState<string | null>(null)
  const [initializing, setInitializing] = useState(true)

  const applyToken = useCallback((accessToken: string, emailHint: string) => {
    setAccessToken(accessToken)
    const claims = decodeJwt(accessToken)
    setEmail(claims?.email ?? emailHint)
    setRole(claims?.role ?? null)
  }, [])

  // On mount, try to re-hydrate the session from the HttpOnly refresh_token
  // cookie. Required because the access token lives in memory only — any
  // full-page reload (notably the Stripe checkout redirect) drops it.
  //
  // The ref guard ensures this fires exactly once per app lifetime, even
  // under React.StrictMode's dev-mode double-invoke. Without it, two
  // refresh calls race with the same cookie value, the backend marks the
  // token Used after the first, the second hits the "token reuse" branch
  // and calls RevokeAllUserTokens — logging the user out everywhere.
  const refreshFired = useRef(false)
  useEffect(() => {
    if (refreshFired.current) return
    refreshFired.current = true

    api
      .refresh()
      .then((res) => {
        applyToken(res.access_token, '')
      })
      .catch((e) => {
        // 401 just means no/expired cookie — user is a guest. Anything else
        // we surface to the console for debugging but don't block the app.
        if (!(e instanceof ApiError) || e.status !== 401) {
          console.warn('auth refresh failed:', e)
        }
      })
      .finally(() => {
        setInitializing(false)
      })
  }, [applyToken])

  const login = useCallback(
    async (e: string, p: string, totp?: string) => {
      const res = await api.login(e, p, totp)
      applyToken(res.access_token, e)
    },
    [applyToken],
  )

  const loginWithToken = useCallback(
    (accessToken: string, emailHint: string) => {
      applyToken(accessToken, emailHint)
    },
    [applyToken],
  )

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } catch {
      // Best-effort: even if the server call fails, clear local state so
      // the UI reflects logged-out immediately.
    }
    setAccessToken(null)
    setEmail(null)
    setRole(null)
  }, [])

  const value = useMemo<AuthState>(
    () => ({
      isAuthed: email !== null,
      initializing,
      email,
      role,
      login,
      loginWithToken,
      logout,
    }),
    [email, initializing, role, login, loginWithToken, logout],
  )

  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}
