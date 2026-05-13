import { useCallback, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { SidePanelCtx } from './sidePanelCtx'
import type { SidePanelState } from './sidePanelCtx'
import { SidePanel } from './SidePanel'

export function SidePanelProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)

  const open = useCallback((trigger?: HTMLElement | null) => {
    triggerRef.current = trigger ?? null
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    const t = triggerRef.current
    if (t) {
      window.setTimeout(() => {
        try {
          t.focus()
        } catch {
          // Trigger unmounted — drop silently.
        }
      }, 50)
    }
  }, [])

  const toggle = useCallback(
    (trigger?: HTMLElement | null) => {
      if (isOpen) close()
      else open(trigger)
    },
    [isOpen, open, close],
  )

  const value = useMemo<SidePanelState>(
    () => ({ isOpen, open, close, toggle }),
    [isOpen, open, close, toggle],
  )

  return (
    <SidePanelCtx.Provider value={value}>
      {children}
      <SidePanel />
    </SidePanelCtx.Provider>
  )
}
