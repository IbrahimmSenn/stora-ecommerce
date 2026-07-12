import { Link, Navigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Seo } from '../components/Seo'
import { SavedAddresses } from './SavedAddresses'
import { ProfileSection } from './ProfileSection'
import { PasswordSection } from './PasswordSection'

export function AccountPage() {
  const { isAuthed, initializing, email, name, role } = useAuth()
  // Wait out the mount-time token refresh, otherwise a direct visit to
  // /account bounces to /login before the session re-hydrates.
  if (initializing) return null
  if (!isAuthed) return <Navigate to="/login" replace />

  const staff = role && role !== 'customer'

  return (
    <Page width="max-w-3xl">
      <Seo title="Your account" noindex />
      <Masthead
        eyebrow="Account"
        title={name || email || 'Your account'}
        caption={
          staff
            ? `Staff account — ${role}.`
            : 'Manage your profile, security, and saved details below.'
        }
      />

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">Profile</div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink font-bold">Your details</h2>
          <p className="text-sm text-ink-soft max-w-[55ch]">
            Your name appears in the menu and on your account. Your email is
            your sign-in and where order updates go.
          </p>
          <div className="pt-3">
            <ProfileSection />
          </div>
        </div>
      </section>

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">Password</div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink font-bold">Change password</h2>
          <p className="text-sm text-ink-soft max-w-[55ch]">
            Pick a new password. Once changed, other signed-in devices are
            logged out.
          </p>
          <div className="pt-3">
            <PasswordSection />
          </div>
        </div>
      </section>

      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">Security</div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink font-bold">
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
        <div className="uc-tight text-[0.7rem] text-ink-faint">Orders</div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink font-bold">
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
        <div className="uc-tight text-[0.7rem] text-ink-faint">Addresses</div>
        <SavedAddresses />
      </section>

      {import.meta.env.DEV && (
      <section className="grid grid-cols-1 md:grid-cols-[12rem_1fr] gap-y-6 gap-x-12 py-8 border-t border-rule">
        <div className="uc-tight text-[0.7rem] text-ink-faint">Developer</div>
        <div className="space-y-2">
          <h2 className="font-display text-xl text-ink font-bold">
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
      )}
    </Page>
  )
}
