import { useContext } from 'react'
import { CartCtx } from './cartCtx'
import type { CartState } from './cartCtx'

export function useCart(): CartState {
  const ctx = useContext(CartCtx)
  if (!ctx) throw new Error('useCart must be used inside CartProvider')
  return ctx
}
