/* ThemeToggle.tsx — icon-only Sun/Moon switch.
 *
 * Shown inside the side panel (the only surface that hosts theme control).
 * Sun icon = currently light, click to go dark. Moon icon = currently dark,
 * click to go light. Honors prefers-reduced-motion via the colour-only state
 * change — no rotation or morph animation.
 */
import { useTheme } from '../lib/theme'
import { Moon, Sun } from './icons'

export function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const isDark = theme === 'dark'
  const Icon = isDark ? Sun : Moon
  const next = isDark ? 'light' : 'dark'

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={`Switch to ${next} mode`}
      className="inline-flex h-9 w-9 items-center justify-center border border-rule text-ink hover:border-accent hover:text-accent transition-colors cursor-pointer"
    >
      <Icon size={16} strokeWidth={1.5} aria-hidden />
    </button>
  )
}
