import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { ToastCtx } from './toastCtx'
import type { ToastMessage, ToastState } from './toastCtx'
import { Toast } from './Toast'

const DISMISS_MS = 3000

export function ToastProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState<ToastMessage | null>(null)
  const idRef = useRef(0)
  const timerRef = useRef<number | null>(null)

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }, [])

  const dismiss = useCallback(() => {
    clearTimer()
    setMessage(null)
  }, [clearTimer])

  const show = useCallback(
    (text: string) => {
      clearTimer()
      idRef.current += 1
      setMessage({ id: idRef.current, text })
      timerRef.current = window.setTimeout(() => {
        setMessage(null)
        timerRef.current = null
      }, DISMISS_MS)
    },
    [clearTimer],
  )

  useEffect(() => () => clearTimer(), [clearTimer])

  const value = useMemo<ToastState>(
    () => ({ message, show, dismiss }),
    [message, show, dismiss],
  )

  return (
    <ToastCtx.Provider value={value}>
      {children}
      <Toast />
    </ToastCtx.Provider>
  )
}
