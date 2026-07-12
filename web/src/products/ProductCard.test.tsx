import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'

import { ProductCard } from './ProductCard'
import type { ProductListItem } from '../lib/api'

function makeProduct(overrides: Partial<ProductListItem> = {}): ProductListItem {
  return {
    id: '11111111-1111-1111-1111-111111111111',
    name: 'Test Lamp',
    price: 10000,
    sale_price: null,
    stock_quantity: 12,
    primary_image: null,
    avg_rating: 0,
    review_count: 0,
    ...overrides,
  }
}

function renderCard(props: Parameters<typeof ProductCard>[0]) {
  return render(
    <MemoryRouter>
      <ProductCard {...props} />
    </MemoryRouter>,
  )
}

describe('ProductCard', () => {
  it('shows the plain price and no sale chrome when not on sale', () => {
    renderCard({ product: makeProduct() })
    expect(screen.getByText('$100.00')).toBeInTheDocument()
    expect(screen.queryByText(/save/i)).toBeNull()
    expect(screen.queryByText(/-\d+%/)).toBeNull()
  })

  it('shows sale price, save chip, struck normal price, and discount flag on sale', () => {
    renderCard({ product: makeProduct({ sale_price: 7500 }) })
    expect(screen.getByText('$75.00')).toBeInTheDocument()
    expect(screen.getByText('Save $25.00')).toBeInTheDocument()
    expect(screen.getByText('$100.00')).toBeInTheDocument() // struck "Normal" price
    expect(screen.getByText('-25%')).toBeInTheDocument()
  })

  it('disables quick-add and shows out of stock when quantity is zero', () => {
    const onAdd = vi.fn()
    renderCard({ product: makeProduct({ stock_quantity: 0 }), onAdd })
    expect(screen.getByText(/out of stock/i)).toBeInTheDocument()
    const btn = screen.getByRole('button', { name: /unavailable/i })
    expect(btn).toBeDisabled()
  })

  it('fires onAdd from the quick-add button', async () => {
    const onAdd = vi.fn()
    renderCard({ product: makeProduct(), onAdd })
    await userEvent.click(screen.getByRole('button', { name: /add test lamp to cart/i }))
    expect(onAdd).toHaveBeenCalledTimes(1)
  })

  it('renders the bestseller badge when requested', () => {
    renderCard({ product: makeProduct(), badge: 'bestseller' })
    expect(screen.getByText(/bestseller/i)).toBeInTheDocument()
  })
})
