import { useCallback, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, setAccessToken } from '../lib/api'
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

  const applyToken = useCallback((accessToken: string, emailHint: string) => {
    setAccessToken(accessToken)
    const claims = decodeJwt(accessToken)
    setEmail(claims?.email ?? emailHint)
    setRole(claims?.role ?? null)
  }, [])

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

  const logout = useCallback(() => {
    setAccessToken(null)
    setEmail(null)
    setRole(null)
  }, [])

  const value = useMemo<AuthState>(
    () => ({
      isAuthed: email !== null,
      email,
      role,
      login,
      loginWithToken,
      logout,
    }),
    [email, role, login, loginWithToken, logout],
  )

  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}
