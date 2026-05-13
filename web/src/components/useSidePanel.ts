import { useContext } from 'react'
import { SidePanelCtx } from './sidePanelCtx'
import type { SidePanelState } from './sidePanelCtx'

export function useSidePanel(): SidePanelState {
  const ctx = useContext(SidePanelCtx)
  if (!ctx) throw new Error('useSidePanel must be used inside SidePanelProvider')
  return ctx
}
