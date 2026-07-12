import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'

import { NavSearch } from './NavSearch'
import { api } from '../lib/api'
import type { ProductSuggestion } from '../lib/api'

describe('NavSearch suggestions', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows thumbnail, name, and prices for suggestions', async () => {
    const suggestions: ProductSuggestion[] = [
      {
        id: '11111111-1111-1111-1111-111111111111',
        name: 'Chanel Coco Noir',
        price: 12999,
        sale_price: 9999,
        image_url: '/products/chanel-1.webp',
      },
      {
        id: '22222222-2222-2222-2222-222222222222',
        name: 'Chair',
        price: 4900,
        sale_price: null,
        image_url: null,
      },
    ]
    vi.spyOn(api, 'suggestProducts').mockResolvedValue(suggestions)

    render(
      <MemoryRouter>
        <NavSearch prominent />
      </MemoryRouter>,
    )

    const input = screen.getByRole('combobox', { name: /search products/i })
    await userEvent.type(input, 'cha')

    // Debounced fetch → wait for the option rows.
    const option = await screen.findByRole('option', { name: /chanel coco noir/i })
    expect(option).toBeInTheDocument()
    expect(screen.getByText('$99.99')).toBeInTheDocument() // sale price
    expect(screen.getByText('$129.99')).toBeInTheDocument() // struck normal price
    expect(screen.getByText('$49.00')).toBeInTheDocument() // plain price
  })
})
