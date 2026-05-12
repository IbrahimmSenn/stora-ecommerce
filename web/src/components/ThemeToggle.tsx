import { useTheme } from '../lib/theme'

export function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const label = theme === 'dark' ? 'Light' : 'Dark'
  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={`Switch to ${label.toLowerCase()} mode`}
      className="uc-tight text-[0.7rem] text-ink-soft hover:text-ink transition-colors px-2 py-1 -mr-2"
    >
      {label}
    </button>
  )
}
