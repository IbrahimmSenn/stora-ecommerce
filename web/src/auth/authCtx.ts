import { createContext } from 'react'

export type AuthState = {
  isAuthed: boolean
  email: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

export const AuthCtx = createContext<AuthState | null>(null)
