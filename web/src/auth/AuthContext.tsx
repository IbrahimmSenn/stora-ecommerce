import { useCallback, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, setAccessToken } from '../lib/api'
import { AuthCtx } from './authCtx'
import type { AuthState } from './authCtx'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [email, setEmail] = useState<string | null>(null)

  const login = useCallback(async (e: string, p: string) => {
    const res = await api.login(e, p)
    setAccessToken(res.access_token)
    setEmail(e)
  }, [])

  const logout = useCallback(() => {
    setAccessToken(null)
    setEmail(null)
  }, [])

  const value = useMemo<AuthState>(
    () => ({ isAuthed: email !== null, email, login, logout }),
    [email, login, logout],
  )

  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}
