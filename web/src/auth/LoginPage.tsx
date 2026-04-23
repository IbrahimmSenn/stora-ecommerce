import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from './useAuth'
import { useCart } from '../cart/useCart'
import { MergePromptModal } from './MergePromptModal'
import { ApiError } from '../lib/api'
import type { Cart } from '../lib/api'

export function LoginPage() {
  const { login } = useAuth()
  const { refresh, fetchMergeStatus } = useCart()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState<{ guest: Cart; user: Cart } | null>(
    null,
  )

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(email, password)
      const status = await fetchMergeStatus()
      if (status.conflict && status.guest_cart && status.user_cart) {
        setConflict({ guest: status.guest_cart, user: status.user_cart })
        return
      }
      await refresh()
      navigate('/cart')
    } catch (err) {
      if (err instanceof ApiError) setError(err.message)
      else setError('login failed')
    } finally {
      setBusy(false)
    }
  }

  async function handleResolved() {
    setConflict(null)
    await refresh()
    navigate('/cart')
  }

  return (
    <div className="max-w-sm mx-auto mt-16">
      <h1 className="text-2xl font-semibold mb-6">Log in</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        <label className="block">
          <span className="block text-sm mb-1">Email</span>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full border px-3 py-2 rounded"
            autoComplete="email"
          />
        </label>
        <label className="block">
          <span className="block text-sm mb-1">Password</span>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full border px-3 py-2 rounded"
            autoComplete="current-password"
          />
        </label>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <button
          type="submit"
          disabled={busy}
          className="w-full bg-black text-white py-2 rounded disabled:opacity-50"
        >
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>

      {conflict && (
        <MergePromptModal
          guestCart={conflict.guest}
          userCart={conflict.user}
          onResolved={handleResolved}
        />
      )}
    </div>
  )
}
