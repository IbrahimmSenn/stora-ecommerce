import { useCallback, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { CartPanelCtx } from './cartPanelCtx'
import type { AddedItem, CartPanelState } from './cartPanelCtx'
import { CartPanel } from './CartPanel'

export function CartPanelProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false)
  const [added, setAdded] = useState<AddedItem | null>(null)
  const triggerRef = useRef<HTMLElement | null>(null)

  const openWith = useCallback(
    (item: AddedItem, trigger?: HTMLElement | null) => {
      triggerRef.current = trigger ?? null
      setAdded(item)
      setIsOpen(true)
    },
    [],
  )

  const open = useCallback((trigger?: HTMLElement | null) => {
    triggerRef.current = trigger ?? null
    setAdded(null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    // Return focus to the trigger once the slide-out has begun. 50ms is
    // enough that the panel is visibly receding before focus shifts.
    const t = triggerRef.current
    if (t) {
      window.setTimeout(() => {
        try {
          t.focus()
        } catch {
          // Element may have unmounted (route change). Drop silently.
        }
      }, 50)
    }
  }, [])

  const value = useMemo<CartPanelState>(
    () => ({ isOpen, added, openWith, open, close }),
    [isOpen, added, openWith, open, close],
  )

  return (
    <CartPanelCtx.Provider value={value}>
      {children}
      <CartPanel />
    </CartPanelCtx.Provider>
  )
}
