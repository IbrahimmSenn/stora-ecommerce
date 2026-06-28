/* PasswordChecklist.tsx — live password-rule checklist.
 *
 * Each rule shows a check when met (positive colour) and a neutral dot when
 * not. Shown beneath the password field on register / reset. Announced politely
 * so screen-reader users hear rule changes as they type.
 */
import { passwordCriteria } from './passwordCriteria'
import { Check } from '../components/icons'

export function PasswordChecklist({ password }: { password: string }) {
  return (
    <ul
      aria-label="Password requirements"
      aria-live="polite"
      className="rounded-lg border border-rule bg-sunken px-4 py-3 space-y-1.5"
    >
      {passwordCriteria.map((c) => {
        const met = c.test(password)
        return (
          <li
            key={c.label}
            className={`flex items-center gap-2 text-sm transition-colors ${
              met ? 'text-positive' : 'text-ink-faint'
            }`}
          >
            <span
              className={`inline-flex h-4 w-4 items-center justify-center rounded-full shrink-0 ${
                met ? 'bg-positive text-surface' : 'border border-rule-strong'
              }`}
            >
              {met && <Check size={11} strokeWidth={3} aria-hidden />}
            </span>
            <span>{c.label}</span>
            <span className="sr-only">{met ? '— met' : '— not met'}</span>
          </li>
        )
      })}
    </ul>
  )
}
