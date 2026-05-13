import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'
import type { ReactNode } from 'react'
import { api, ApiError } from '../lib/api'
import type { Cart } from '../lib/api'
import { useAuth } from '../auth/useAuth'
import { CartCtx } from './cartCtx'
import type { CartState } from './cartCtx'

export function CartProvider({ children }: { children: ReactNode }) {
  const { initializing: authInitializing, email } = useAuth()
  const [cart, setCart] = useState<Cart | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async (fn: () => Promise<Cart>) => {
    setError(null)
    try {
      const next = await fn()
      setCart(next)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'cart request failed')
      throw e
    }
  }, [])

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const c = await api.getCart()
      setCart(c)
      setError(null)
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        setCart(null)
      } else {
        setError(e instanceof Error ? e.message : 'failed to load cart')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    // Wait for AuthProvider's mount-time refresh to settle, then fetch the
    // cart. Re-fetches when `email` changes (login/logout) so the navbar
    // reflects the correct cart for the current identity.
    if (authInitializing) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- post-auth fetch
    void refresh()
  }, [refresh, authInitializing, email])

  const addItem = useCallback(
    (productId: string, quantity: number) =>
      run(() => api.addItem(productId, quantity)),
    [run],
  )
  const updateItem = useCallback(
    (productId: string, quantity: number) =>
      run(() => api.updateItem(productId, quantity)),
    [run],
  )
  const removeItem = useCallback(
    (productId: string) => run(() => api.removeItem(productId)),
    [run],
  )
  const clear = useCallback(async () => {
    setError(null)
    await api.clearCart()
    await refresh()
  }, [refresh])

  const fetchMergeStatus = useCallback(() => api.mergeStatus(), [])

  const resolveMerge = useCallback(
    async (strategy: 'guest' | 'user') => {
      const next = await api.merge(strategy)
      setCart(next)
    },
    [],
  )

  const itemCount = useMemo(
    () => cart?.items.reduce((sum, it) => sum + it.quantity, 0) ?? 0,
    [cart],
  )

  const value = useMemo<CartState>(
    () => ({
      cart,
      loading,
      error,
      itemCount,
      refresh,
      addItem,
      updateItem,
      removeItem,
      clear,
      fetchMergeStatus,
      resolveMerge,
    }),
    [
      cart,
      loading,
      error,
      itemCount,
      refresh,
      addItem,
      updateItem,
      removeItem,
      clear,
      fetchMergeStatus,
      resolveMerge,
    ],
  )

  return <CartCtx.Provider value={value}>{children}</CartCtx.Provider>
}
