import { createContext } from 'react'

export type AuthState = {
  isAuthed: boolean
  email: string | null
  role: string | null
  login: (email: string, password: string, totp?: string) => Promise<void>
  loginWithToken: (accessToken: string, email: string) => void
  logout: () => void
}

export const AuthCtx = createContext<AuthState | null>(null)
