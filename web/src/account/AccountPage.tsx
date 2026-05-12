import { Link, Navigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'

export function AccountPage() {
  const { isAuthed, email, role } = useAuth()
  if (!isAuthed) return <Navigate to="/login" replace />

  return (
    <Page width="max-w-3xl">
      <Masthead
        number="01"
        eyebrow="Account"
        title={email ?? 'Signed in.'}
        caption={
          role
            ? `Signed in as ${role}.`
            : 'Manage your security and integrations below.'
        }
      />

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">
          <span className="tnum">01</span> · Security
        </div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink" style={{ fontVariationSettings: '"wght" 500' }}>
            Two-factor authentication
          </h2>
          <p className="text-sm text-ink-soft max-w-[55ch]">
            Adds a TOTP step to every log-in. Enable once on a phone authenticator
            (Google Authenticator, 1Password, Authy). Eight recovery codes are
            issued at setup time.
          </p>
          <div className="flex gap-4 pt-3">
            <Link
              to="/account/2fa/setup"
              className="text-sm text-ink underline underline-offset-4"
            >
              Set up two-factor.
            </Link>
            <Link
              to="/account/2fa/disable"
              className="text-sm text-ink-soft hover:text-ink"
            >
              Disable.
            </Link>
          </div>
        </div>
      </section>

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">
          <span className="tnum">02</span> · Orders
        </div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink" style={{ fontVariationSettings: '"wght" 500' }}>
            Order history
          </h2>
          <p className="text-sm text-ink-soft">
            View past orders, track current ones, request cancellation.
          </p>
          <Link
            to="/orders"
            className="text-sm text-ink underline underline-offset-4 inline-block pt-3"
          >
            Open order history.
          </Link>
        </div>
      </section>

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">
          <span className="tnum">03</span> · Developer
        </div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink" style={{ fontVariationSettings: '"wght" 500' }}>
            Token rotation tester
          </h2>
          <p className="text-sm text-ink-soft max-w-[55ch]">
            Demonstrates refresh-token rotation and replay detection. Internal
            surface for technical reviewers.
          </p>
          <Link
            to="/dev/tokens"
            className="text-sm text-ink underline underline-offset-4 inline-block pt-3"
          >
            Open token tester.
          </Link>
        </div>
      </section>
    </Page>
  )
}
