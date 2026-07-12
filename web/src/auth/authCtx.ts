import { createContext } from 'react'

export type AuthState = {
  isAuthed: boolean
  /** True until the mount-time refresh attempt completes. Pages that
   *  depend on auth (order detail, account) should wait on this before
   *  firing their first fetch, otherwise a full-page reload will race
   *  the refresh and call the API unauthenticated. */
  initializing: boolean
  email: string | null
  /** Display name from the profile; null until set by the user. */
  name: string | null
  role: string | null
  login: (email: string, password: string, totp?: string) => Promise<void>
  loginWithToken: (accessToken: string, email: string) => void
  logout: () => Promise<void>
  /** Re-fetch /me so profile edits update the nav immediately. */
  refreshMe: () => Promise<void>
}

export const AuthCtx = createContext<AuthState | null>(null)
