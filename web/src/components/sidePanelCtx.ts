import { createContext } from 'react'

export type SidePanelState = {
  isOpen: boolean
  /** Open the panel. Stores the trigger element so focus returns on close. */
  open: (trigger?: HTMLElement | null) => void
  close: () => void
  toggle: (trigger?: HTMLElement | null) => void
}

export const SidePanelCtx = createContext<SidePanelState | null>(null)
