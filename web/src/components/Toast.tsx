/* Toast.tsx — single non-blocking notification, bottom-right.
 *
 * Used for quick-add confirmation from product cards. Replace-only: a new
 * show() cancels the previous timer and swaps content. The message id is the
 * React key so a fresh show() unmounts/remounts the toast and the enter
 * keyframe replays. Transform + opacity only; the reduced-motion override in
 * tokens.css collapses the animation duration to 0ms.
 */
import { createPortal } from 'react-dom'
import { useToast } from './useToast'

export function Toast() {
  const { message, dismiss } = useToast()

  if (!message) return null

  return createPortal(
    <div
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className="fixed bottom-6 right-6 z-50 pointer-events-none"
    >
      <button
        key={message.id}
        type="button"
        onClick={dismiss}
        className="pointer-events-auto cursor-pointer text-left bg-surface border border-rule px-5 py-3 max-w-sm shadow-[0_1px_2px_oklch(0.18_0.01_25/0.04),0_8px_24px_oklch(0.18_0.01_25/0.08)]"
        style={{
          animation: 'toast-in var(--duration-fast) var(--ease-out-quart) both',
        }}
      >
        <span className="text-sm text-ink">{message.text}</span>
      </button>
    </div>,
    document.body,
  )
}
