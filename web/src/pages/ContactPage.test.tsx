import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

import { ContactPage } from './ContactPage'
import { api } from '../lib/api'

function renderPage() {
  return render(
    <MemoryRouter>
      <ContactPage />
    </MemoryRouter>,
  )
}

describe('ContactPage', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('shows a validation error for whitespace-only input', () => {
    renderPage()
    // A valid email passes the HTML5 constraint so the form actually submits;
    // the whitespace-only text fields must then be rejected by the component's
    // own trim() validation.
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'jane@example.com' } })
    for (const name of ['Name', 'Subject', 'Message']) {
      fireEvent.change(screen.getByLabelText(name), { target: { value: '   ' } })
    }
    fireEvent.click(screen.getByRole('button', { name: /send message/i }))
    expect(screen.getByText(/please fill in every field/i)).toBeTruthy()
  })

  it('submits the form and shows the success state', async () => {
    const spy = vi.spyOn(api, 'submitContact').mockResolvedValue({ message: 'ok' })
    renderPage()

    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Jane' } })
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'jane@example.com' } })
    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'Order question' } })
    fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'Where is my order?' } })
    fireEvent.click(screen.getByRole('button', { name: /send message/i }))

    await waitFor(() => expect(spy).toHaveBeenCalledOnce())
    expect(spy).toHaveBeenCalledWith({
      name: 'Jane',
      email: 'jane@example.com',
      subject: 'Order question',
      message: 'Where is my order?',
    })
    await screen.findByText(/message sent/i)
  })
})
