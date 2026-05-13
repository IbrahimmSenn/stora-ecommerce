import { useContext } from 'react'
import { ToastCtx } from './toastCtx'
import type { ToastState } from './toastCtx'

export function useToast(): ToastState {
  const ctx = useContext(ToastCtx)
  if (!ctx) throw new Error('useToast must be used inside ToastProvider')
  return ctx
}
