import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from './useAuth'
import { useCart } from '../cart/useCart'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'

/**
 * After the OAuth provider redirects back, the backend bounces the user to
 * /auth/oauth/callback#access_token=...&refresh_token=...&email=...
 * This page reads the hash, hands the access token to AuthContext, then
 * navigates home.
 */
export function OAuthCallbackPage() {
  const navigate = useNavigate()
  const { loginWithToken } = useAuth()
  const { refresh } = useCart()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const hash = window.location.hash.replace(/^#/, '')
    if (!hash) {
      setError('No tokens in the redirect URL.')
      return
    }
    const params = new URLSearchParams(hash)
    const access = params.get('access_token')
    const email = params.get('email') ?? ''
    if (!access) {
      setError('Missing access token in the callback.')
      return
    }
    loginWithToken(access, email)
    refresh().finally(() => navigate('/', { replace: true }))
  }, [loginWithToken, navigate, refresh])

  return (
    <Page width="max-w-md">
      <Masthead
        eyebrow="OAuth"
        title={error ? 'Sign-in failed.' : 'Signing you in.'}
        caption={error ?? 'One moment — we are exchanging tokens.'}
      />
    </Page>
  )
}
