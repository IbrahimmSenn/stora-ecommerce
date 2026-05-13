import { useContext } from 'react'
import { CartPanelCtx } from './cartPanelCtx'
import type { CartPanelState } from './cartPanelCtx'

export function useCartPanel(): CartPanelState {
  const ctx = useContext(CartPanelCtx)
  if (!ctx) throw new Error('useCartPanel must be used inside CartPanelProvider')
  return ctx
}
