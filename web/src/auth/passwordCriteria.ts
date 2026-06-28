/* passwordCriteria.ts — the password-strength rules, mirrored from the backend
 * (internal/passwordpolicy). Keep the two in sync. */

export type PasswordCriterion = {
  label: string
  test: (pw: string) => boolean
}

export const passwordCriteria: PasswordCriterion[] = [
  { label: 'At least 8 characters', test: (pw) => pw.length >= 8 },
  { label: 'An uppercase letter (A–Z)', test: (pw) => /[A-Z]/.test(pw) },
  { label: 'A lowercase letter (a–z)', test: (pw) => /[a-z]/.test(pw) },
  { label: 'A number (0–9)', test: (pw) => /[0-9]/.test(pw) },
  { label: 'A symbol (!@#$%…)', test: (pw) => /[^A-Za-z0-9\s]/.test(pw) },
]

/** True when the password satisfies every rule. */
export function passwordIsStrong(pw: string): boolean {
  return passwordCriteria.every((c) => c.test(pw))
}
