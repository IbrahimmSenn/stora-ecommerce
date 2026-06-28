import type { ReactNode } from 'react'
import { Reveal } from '../lib/motion'

type PageProps = {
  children: ReactNode
  /** Tailwind max-width utility. Default 'max-w-5xl' for content; auth/admin
   *  pages typically pass 'max-w-md' or 'max-w-3xl'. */
  width?: string
  className?: string
}

/**
 * Page is the route shell: max-width wrapper + first-paint Reveal applied to
 * the immediate children. Use as the outermost element of every route.
 */
export function Page({ children, width = 'max-w-7xl', className = '' }: PageProps) {
  return (
    <Reveal
      stagger={70}
      className={`${width} mx-auto px-4 lg:px-8 py-8 lg:py-12 ${className}`}
    >
      {children}
    </Reveal>
  )
}
