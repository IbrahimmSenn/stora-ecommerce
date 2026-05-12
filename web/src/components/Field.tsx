import type { InputHTMLAttributes, ReactNode } from 'react'
import { useId } from 'react'

type FieldProps = {
  label: ReactNode
  hint?: ReactNode
  error?: ReactNode
} & InputHTMLAttributes<HTMLInputElement>

export function Field({ label, hint, error, className = '', ...rest }: FieldProps) {
  const id = useId()
  return (
    <label htmlFor={id} className="block">
      <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
        {label}
      </span>
      <input
        id={id}
        className={`w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink placeholder-ink-faint transition-colors ${className}`}
        style={{ borderRadius: 0 }}
        {...rest}
      />
      {hint && !error && (
        <span className="block text-xs text-ink-faint mt-1.5">{hint}</span>
      )}
      {error && (
        <span className="block text-xs text-accent mt-1.5">{error}</span>
      )}
    </label>
  )
}
