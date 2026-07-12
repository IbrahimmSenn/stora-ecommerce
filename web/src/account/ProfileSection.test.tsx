import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ProfileSection } from './ProfileSection'
import { api, ApiError } from '../lib/api'
import type { Me } from '../lib/api'

const refreshMe = vi.fn(async () => {})

vi.mock('../auth/useAuth', () => ({
  useAuth: () => ({ initializing: false, refreshMe }),
}))

const me: Me = {
  id: '11111111-1111-1111-1111-111111111111',
  email: 'ada@example.com',
  name: 'Ada',
  role: 'customer',
  created_at: '2026-01-15T10:00:00Z',
}

describe('ProfileSection', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    refreshMe.mockClear()
  })

  it('loads and shows the current profile', async () => {
    vi.spyOn(api, 'me').mockResolvedValue(me)

    render(<ProfileSection />)

    expect(await screen.findByLabelText(/name/i)).toHaveValue('Ada')
    expect(screen.getByText('ada@example.com')).toBeInTheDocument()
    expect(screen.getByText(/member since/i)).toBeInTheDocument()
  })

  it('saves the name and refreshes the auth context', async () => {
    vi.spyOn(api, 'me').mockResolvedValue(me)
    const update = vi
      .spyOn(api, 'updateProfile')
      .mockResolvedValue({ ...me, name: 'Ada Lovelace' })

    render(<ProfileSection />)
    const input = await screen.findByLabelText(/name/i)
    await userEvent.clear(input)
    await userEvent.type(input, 'Ada Lovelace')
    await userEvent.click(screen.getByRole('button', { name: /save profile/i }))

    await waitFor(() => expect(update).toHaveBeenCalledWith('Ada Lovelace'))
    expect(await screen.findByText(/profile saved/i)).toBeInTheDocument()
    expect(refreshMe).toHaveBeenCalled()
  })

  it('shows the server error message when saving fails', async () => {
    vi.spyOn(api, 'me').mockResolvedValue(me)
    vi.spyOn(api, 'updateProfile').mockRejectedValue(
      new ApiError(400, 'name must be 100 characters or fewer'),
    )

    render(<ProfileSection />)
    await screen.findByLabelText(/name/i)
    await userEvent.click(screen.getByRole('button', { name: /save profile/i }))

    expect(
      await screen.findByText(/name must be 100 characters or fewer/i),
    ).toBeInTheDocument()
    expect(refreshMe).not.toHaveBeenCalled()
  })
})
