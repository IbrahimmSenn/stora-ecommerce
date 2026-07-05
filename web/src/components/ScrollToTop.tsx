/* ScrollToTop.tsx — a fixed bottom-right button that fades in once the page is
 * scrolled past the masthead, returning the viewer to the top on click.
 */
import { useEffect, useState } from 'react'
import { ArrowUp } from './icons'

const SHOW_AFTER = 500 // px scrolled before the button appears

export function ScrollToTop() {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const onScroll = () => setVisible(window.scrollY > SHOW_AFTER)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  function toTop() {
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.scrollTo({ top: 0, behavior: reduce ? 'auto' : 'smooth' })
  }

  return (
    <button
      type="button"
      onClick={toTop}
      aria-label="Back to top"
      title="Back to top"
      tabIndex={visible ? 0 : -1}
      className={`fixed bottom-6 right-6 z-40 inline-flex h-11 w-11 items-center justify-center bg-accent text-on-accent shadow-[0_6px_20px_oklch(0.2_0.01_265/0.18)] transition-all duration-200 cursor-pointer hover:bg-accent-soft focus:outline-none focus-visible:ring-2 focus-visible:ring-ink ${
        visible ? 'opacity-100 translate-y-0' : 'pointer-events-none opacity-0 translate-y-3'
      }`}
    >
      <ArrowUp size={20} strokeWidth={2} aria-hidden />
    </button>
  )
}
