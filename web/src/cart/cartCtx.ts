import { createContext } from 'react'
import type { Cart, MergeStatus } from '../lib/api'

export type CartState = {
  cart: Cart | null
  loading: boolean
  error: string | null
  itemCount: number
  refresh: () => Promise<void>
  addItem: (productId: string, quantity: number) => Promise<void>
  updateItem: (productId: string, quantity: number) => Promise<void>
  removeItem: (productId: string) => Promise<void>
  clear: () => Promise<void>
  fetchMergeStatus: () => Promise<MergeStatus>
  resolveMerge: (strategy: 'guest' | 'user') => Promise<void>
}

export const CartCtx = createContext<CartState | null>(null)
