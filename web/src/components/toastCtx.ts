import { createContext } from 'react'

export type ToastMessage = {
  /** Monotonic id, increments per show() call. Used as the React key so the
   *  enter transition replays even when the message text is unchanged. */
  id: number
  text: string
}

export type ToastState = {
  /** Current toast, or null when nothing is showing. */
  message: ToastMessage | null
  /** Replace the current toast. Cancels the existing dismiss timer. */
  show: (text: string) => void
  /** Dismiss the current toast immediately. */
  dismiss: () => void
}

export const ToastCtx = createContext<ToastState | null>(null)
