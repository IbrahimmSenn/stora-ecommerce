import type { ButtonHTMLAttributes, ReactNode } from 'react'

type Variant = 'primary' | 'ghost' | 'link'

type ButtonProps = {
  variant?: Variant
  children: ReactNode
} & ButtonHTMLAttributes<HTMLButtonElement>

const styles: Record<Variant, string> = {
  primary:
    'bg-accent text-on-accent hover:bg-accent-soft transition-colors px-5 py-2.5 text-sm tracking-[0.01em] disabled:opacity-50 disabled:cursor-not-allowed',
  ghost:
    'bg-transparent text-ink border border-rule-strong hover:border-ink hover:bg-sunken transition-colors px-5 py-2.5 text-sm disabled:opacity-50 disabled:cursor-not-allowed',
  link: 'bg-transparent text-ink-soft hover:text-ink underline underline-offset-4 text-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed',
}

export function Button({
  variant = 'primary',
  className = '',
  type = 'button',
  children,
  ...rest
}: ButtonProps) {
  return (
    <button type={type} className={`${styles[variant]} ${className}`} {...rest}>
      {children}
    </button>
  )
}
