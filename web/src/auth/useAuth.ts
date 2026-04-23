import { useContext } from 'react'
import { AuthCtx } from './authCtx'
import type { AuthState } from './authCtx'

export function useAuth(): AuthState {
  const ctx = useContext(AuthCtx)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
