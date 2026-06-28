import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { StarRating, StarRatingInput } from './StarRating'

describe('StarRating', () => {
  it('exposes the numeric value in its accessible label', () => {
    render(<StarRating value={4.3} />)
    expect(screen.getByRole('img', { name: /4.3 out of 5/i })).toBeTruthy()
  })

  it('renders the review count when provided', () => {
    render(<StarRating value={5} count={12} />)
    expect(screen.getByText('(12)')).toBeTruthy()
  })
})

describe('StarRatingInput', () => {
  it('reports the chosen rating', () => {
    const onChange = vi.fn()
    render(<StarRatingInput value={0} onChange={onChange} />)
    fireEvent.click(screen.getByRole('radio', { name: /4 stars/i }))
    expect(onChange).toHaveBeenCalledWith(4)
  })

  it('marks the current value as checked', () => {
    render(<StarRatingInput value={3} onChange={() => {}} />)
    expect(screen.getByRole('radio', { name: /3 stars/i }).getAttribute('aria-checked')).toBe('true')
  })
})
