import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

import { RecommendationsRail } from './RecommendationsRail'
import { api } from '../lib/api'
import type { ProductListItem } from '../lib/api'

describe('RecommendationsRail', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders nothing while loading and nothing when the API returns []', async () => {
    const spy = vi
      .spyOn(api, 'recommendations')
      .mockResolvedValue({ items: [] })

    const { container } = render(
      <MemoryRouter>
        <RecommendationsRail cartVersion={1} />
      </MemoryRouter>,
    )

    // Loading: nothing rendered yet.
    expect(container.firstChild).toBeNull()

    await waitFor(() => expect(spy).toHaveBeenCalled())

    // Empty payload: still hidden.
    expect(screen.queryByText(/for you/i)).toBeNull()
  })

  it('renders one tile per recommendation with name, price, and a link', async () => {
    const items: ProductListItem[] = [
      {
        id: '11111111-1111-1111-1111-111111111111',
        name: 'Hanging Lantern',
        price: 4200,
        stock_quantity: 10,
        primary_image: 'https://example.com/lantern.jpg',
        avg_rating: 0,
        review_count: 0,
      },
      {
        id: '22222222-2222-2222-2222-222222222222',
        name: 'Glass Vase',
        price: 9900,
        stock_quantity: 2,
        primary_image: null,
        avg_rating: 0,
        review_count: 0,
      },
    ]
    vi.spyOn(api, 'recommendations').mockResolvedValue({ items })

    render(
      <MemoryRouter>
        <RecommendationsRail cartVersion={1} />
      </MemoryRouter>,
    )

    // findByRole waits for the first render with content.
    const lanternLink = await screen.findByRole('link', {
      name: /hanging lantern/i,
    })
    const vaseLink = screen.getByRole('link', { name: /glass vase/i })

    expect(lanternLink).toHaveAttribute(
      'href',
      '/product/11111111-1111-1111-1111-111111111111',
    )
    expect(vaseLink).toHaveAttribute(
      'href',
      '/product/22222222-2222-2222-2222-222222222222',
    )
    expect(screen.getByText('$42.00')).toBeInTheDocument()
    expect(screen.getByText('$99.00')).toBeInTheDocument()
  })

  it('survives the API throwing', async () => {
    vi.spyOn(api, 'recommendations').mockRejectedValue(new Error('boom'))

    const { container } = render(
      <MemoryRouter>
        <RecommendationsRail cartVersion={1} />
      </MemoryRouter>,
    )

    // Wait a microtask cycle so the catch handler runs.
    await waitFor(() => expect(container.firstChild).toBeNull())
  })
})
