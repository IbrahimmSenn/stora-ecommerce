/* motion.tsx — small motion primitives.
 *
 * <Reveal> stagger-fades a group of children on mount with translateY 8 → 0
 * and opacity 0 → 1. Uses ease-out-quart. Honors prefers-reduced-motion by
 * mounting instantly. Pure CSS transitions; no animation library.
 */
import {
  Children,
  isValidElement,
  cloneElement,
  useEffect,
  useState,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  type ReactElement,
} from 'react'

export function useReducedMotion(): boolean {
  const [prefers, setPrefers] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  })

  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    const handler = (e: MediaQueryListEvent) => setPrefers(e.matches)
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  return prefers
}

type RevealProps = {
  children: ReactNode
  /** ms between successive children. Default 80. */
  stagger?: number
  /** ms before the first child enters. Default 20. */
  delay?: number
  /** Per-child enter duration. Default 480. */
  duration?: number
  className?: string
} & Omit<HTMLAttributes<HTMLDivElement>, 'children'>

/**
 * <Reveal> wraps a small group of siblings and animates each one in turn on
 * mount. Each direct child must accept className + style props (a plain
 * <div>, <h1>, etc.). The component does NOT mount/unmount its children on
 * route change — it's a first-paint flourish.
 */
export function Reveal({
  children,
  stagger = 80,
  delay = 20,
  duration = 480,
  className,
  ...rest
}: RevealProps) {
  const reduced = useReducedMotion()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const t = window.requestAnimationFrame(() => setReady(true))
    return () => window.cancelAnimationFrame(t)
  }, [])

  const items = Children.toArray(children).filter(isValidElement) as ReactElement<{
    className?: string
    style?: CSSProperties
  }>[]

  return (
    <div className={className} {...rest}>
      {items.map((child, i) => {
        const offset = ready ? 0 : 8
        const opacity = ready ? 1 : 0
        const transitionDelay = reduced ? 0 : delay + i * stagger
        const style: CSSProperties = {
          ...(child.props.style ?? {}),
          opacity,
          transform: `translateY(${offset}px)`,
          transition: reduced
            ? 'none'
            : `opacity ${duration}ms var(--ease-out-quart) ${transitionDelay}ms, transform ${duration}ms var(--ease-out-quart) ${transitionDelay}ms`,
          willChange: reduced ? undefined : 'opacity, transform',
        }
        return cloneElement(child, { ...child.props, style, key: child.key ?? i })
      })}
    </div>
  )
}
