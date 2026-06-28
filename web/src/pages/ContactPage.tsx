import { useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { Seo } from '../components/Seo'
import { api, ApiError } from '../lib/api'

export function ContactPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !email.trim() || !subject.trim() || !message.trim()) {
      setError('Please fill in every field.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.submitContact({ name, email, subject, message })
      setSent(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not send your message.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-3xl">
      <Seo
        title="Contact us"
        description="Questions about an order, a product, or anything else? Send the Stora team a message and we'll get back to you."
      />
      <Masthead
        eyebrow="Support"
        title="Get in touch."
        caption="Questions about an order, a product, or returns? Send us a note and we'll reply by email."
      />

      {sent ? (
        <div role="status" className="border-t border-rule pt-8">
          <h2 className="font-display text-2xl text-ink font-bold mb-3">Message sent.</h2>
          <p className="text-ink-soft">
            Thanks for reaching out — we've got your message and will reply to{' '}
            <span className="text-ink">{email}</span> soon.
          </p>
        </div>
      ) : (
        <form onSubmit={submit} className="flex flex-col gap-6 max-w-xl">
          <Field label="Name" required value={name} onChange={(e) => setName(e.target.value)} />
          <Field label="Email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          <Field label="Subject" required value={subject} onChange={(e) => setSubject(e.target.value)} />
          <label className="block">
            <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">Message</span>
            <textarea
              required
              rows={6}
              maxLength={4000}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors resize-y outline-none"
              style={{ borderRadius: 0 }}
            />
          </label>
          {error && <p className="text-sm text-accent" role="alert">{error}</p>}
          <div>
            <Button type="submit" disabled={busy}>
              {busy ? 'Sending.' : 'Send message'}
            </Button>
          </div>
        </form>
      )}
    </Page>
  )
}
