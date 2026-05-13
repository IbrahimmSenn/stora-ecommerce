/* NavSearch.tsx — minimal expanding search with a suggest dropdown.
 *
 * Collapsed: narrow input with a thin rule underline, no background.
 * Focused: expands width via transform-free width transition (animating
 * width is layout-affecting, so we transition the wrapper width which is
 * cheap on a static parent flex row — acceptable here given the nav row
 * has stable height).
 *
 * Typing fires debounced /products/suggest. Arrow keys traverse the list.
 * Enter on a selected suggestion navigates to /product/<id>; Enter with
 * no selection navigates to /?q=<query>. Esc clears + closes.
 */
import { useEffect, useId, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import type { ProductSuggestion } from '../lib/api'
import { Search } from './icons'

const DEBOUNCE_MS = 200

export function NavSearch() {
  const navigate = useNavigate()
  const id = useId()
  const wrapperRef = useRef<HTMLDivElement | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [query, setQuery] = useState('')
  const [focused, setFocused] = useState(false)
  const [suggestions, setSuggestions] = useState<ProductSuggestion[]>([])
  const [activeIndex, setActiveIndex] = useState(-1)

  // Debounced fetch. Empty query is handled in the input onChange below so
  // this effect only fires for queries worth a network call.
  useEffect(() => {
    const q = query.trim()
    if (q === '') return
    let cancelled = false
    const t = window.setTimeout(async () => {
      try {
        const next = await api.suggestProducts(q)
        if (cancelled) return
        setSuggestions(next)
        setActiveIndex(-1)
      } catch {
        if (cancelled) return
        setSuggestions([])
      }
    }, DEBOUNCE_MS)
    return () => {
      cancelled = true
      window.clearTimeout(t)
    }
  }, [query])

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const value = e.target.value
    setQuery(value)
    if (value.trim() === '') {
      setSuggestions([])
      setActiveIndex(-1)
    }
  }

  // Close on outside click.
  useEffect(() => {
    if (!focused) return
    function onPointer(e: MouseEvent) {
      if (!wrapperRef.current) return
      if (!wrapperRef.current.contains(e.target as Node)) {
        setFocused(false)
        setActiveIndex(-1)
      }
    }
    document.addEventListener('mousedown', onPointer)
    return () => document.removeEventListener('mousedown', onPointer)
  }, [focused])

  function reset() {
    setQuery('')
    setSuggestions([])
    setActiveIndex(-1)
  }

  function commitQuery() {
    const q = query.trim()
    if (q === '') return
    navigate(`/?q=${encodeURIComponent(q)}`)
    setFocused(false)
    inputRef.current?.blur()
  }

  function commitSuggestion(s: ProductSuggestion) {
    setFocused(false)
    inputRef.current?.blur()
    navigate(`/product/${s.id}`)
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      if (query) {
        reset()
      } else {
        setFocused(false)
        inputRef.current?.blur()
      }
      return
    }
    if (suggestions.length === 0) {
      if (e.key === 'Enter') commitQuery()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((i) => (i + 1) % suggestions.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => (i <= 0 ? suggestions.length - 1 : i - 1))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (activeIndex >= 0 && activeIndex < suggestions.length) {
        commitSuggestion(suggestions[activeIndex])
      } else {
        commitQuery()
      }
    }
  }

  const listOpen = focused && query.trim() !== '' && suggestions.length > 0
  const listboxId = `${id}-listbox`

  return (
    <div ref={wrapperRef} className="relative">
      <div
        className="flex items-center gap-2 border-b transition-all"
        style={{
          width: focused || query ? '20rem' : '12rem',
          borderColor: focused
            ? 'var(--color-rule-strong)'
            : 'var(--color-rule)',
          transition: 'width var(--duration-med) var(--ease-out-quart), border-color 180ms var(--ease-out-quart)',
        }}
      >
        <Search
          size={14}
          strokeWidth={1.5}
          aria-hidden
          className="text-ink-faint shrink-0"
        />
        <input
          ref={inputRef}
          type="text"
          role="combobox"
          aria-expanded={listOpen}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            activeIndex >= 0 ? `${id}-option-${activeIndex}` : undefined
          }
          placeholder="Search"
          value={query}
          onChange={handleChange}
          onFocus={() => setFocused(true)}
          onKeyDown={onKeyDown}
          className="flex-1 bg-transparent text-sm py-2 placeholder:text-ink-faint focus:outline-none"
        />
      </div>

      {listOpen && (
        <ul
          id={listboxId}
          role="listbox"
          className="absolute left-0 right-0 top-full mt-1 z-40 bg-surface border border-rule shadow-[0_8px_24px_oklch(0.18_0.01_25/0.08)]"
        >
          {suggestions.map((s, i) => {
            const active = i === activeIndex
            return (
              <li
                id={`${id}-option-${i}`}
                key={s.id}
                role="option"
                aria-selected={active}
                onMouseEnter={() => setActiveIndex(i)}
                onMouseDown={(e) => {
                  e.preventDefault()
                  commitSuggestion(s)
                }}
                className={`px-4 py-2.5 text-sm cursor-pointer transition-colors ${
                  active ? 'bg-sunken text-ink' : 'text-ink-soft hover:text-ink'
                } ${
                  i < suggestions.length - 1 ? 'border-b border-rule' : ''
                }`}
              >
                {s.name}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
