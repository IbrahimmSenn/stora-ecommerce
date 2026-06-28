import { useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError, formatPrice, type DeliveryOption } from '../lib/api'

type FormState = {
  code: string
  label: string
  price: string // dollars, converted to cents on submit
  eta_label: string
  sort_order: string
  active: boolean
}

const blank: FormState = { code: '', label: '', price: '', eta_label: '', sort_order: '0', active: true }

export function AdminDeliveryPage() {
  const [options, setOptions] = useState<DeliveryOption[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [editId, setEditId] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(blank)
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [rowBusyId, setRowBusyId] = useState<string | null>(null)

  async function refresh() {
    setLoading(true)
    setLoadError(null)
    try {
      setOptions(await api.adminListDeliveryOptions())
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : 'Could not load delivery options.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh()
  }, [])

  function resetForm() {
    setEditId(null)
    setForm(blank)
    setFormError(null)
  }

  function startEdit(o: DeliveryOption) {
    setEditId(o.id)
    setForm({
      code: o.code,
      label: o.label,
      price: (o.price_cents / 100).toFixed(2),
      eta_label: o.eta_label,
      sort_order: String(o.sort_order),
      active: o.active,
    })
    setFormError(null)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const cents = Math.round(parseFloat(form.price) * 100)
    if (!form.label.trim() || Number.isNaN(cents) || cents < 0) {
      setFormError('Label and a non-negative price are required.')
      return
    }
    if (!editId && !form.code.trim()) {
      setFormError('A code is required.')
      return
    }
    setBusy(true)
    setFormError(null)
    try {
      const common = {
        label: form.label.trim(),
        price_cents: cents,
        eta_label: form.eta_label.trim(),
        sort_order: parseInt(form.sort_order, 10) || 0,
        active: form.active,
      }
      if (editId) {
        await api.adminUpdateDeliveryOption(editId, common)
      } else {
        await api.adminCreateDeliveryOption({ code: form.code.trim(), ...common })
      }
      resetForm()
      await refresh()
    } catch (err) {
      setFormError(
        err instanceof ApiError ? err.message : `Could not ${editId ? 'update' : 'create'} option.`,
      )
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete(o: DeliveryOption) {
    if (!window.confirm(`Delete delivery option “${o.label}”?`)) return
    setRowBusyId(o.id)
    try {
      await api.adminDeleteDeliveryOption(o.id)
      if (editId === o.id) resetForm()
      await refresh()
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : 'Could not delete option.')
    } finally {
      setRowBusyId(null)
    }
  }

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Fulfilment"
        title="Delivery options."
        caption="Shipping methods and rates offered at checkout."
      />

      <section className="max-w-2xl mb-16">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
          <span aria-hidden className="text-rule-strong mx-2">/</span>
          {editId ? 'Edit option' : 'New option'}
        </h2>
        <form onSubmit={handleSubmit} className="space-y-6">
          <Field
            label="Code"
            required={!editId}
            disabled={!!editId}
            placeholder="e.g. standard, express, next-day"
            value={form.code}
            onChange={(e) => setForm((f) => ({ ...f, code: e.target.value }))}
            hint={editId ? 'Code is fixed once created (orders reference it).' : 'Lowercase, hyphenated. Used on orders.'}
          />
          <Field
            label="Label"
            required
            value={form.label}
            onChange={(e) => setForm((f) => ({ ...f, label: e.target.value }))}
          />
          <Field
            label="Price (USD)"
            required
            type="number"
            min="0"
            step="0.01"
            value={form.price}
            onChange={(e) => setForm((f) => ({ ...f, price: e.target.value }))}
          />
          <Field
            label="Estimated delivery"
            placeholder="e.g. 5–7 business days"
            value={form.eta_label}
            onChange={(e) => setForm((f) => ({ ...f, eta_label: e.target.value }))}
          />
          <Field
            label="Sort order"
            type="number"
            value={form.sort_order}
            onChange={(e) => setForm((f) => ({ ...f, sort_order: e.target.value }))}
            hint="Lower numbers appear first at checkout."
          />
          <label className="flex items-center gap-3 text-sm text-ink">
            <input
              type="checkbox"
              checked={form.active}
              onChange={(e) => setForm((f) => ({ ...f, active: e.target.checked }))}
              className="accent-current"
            />
            Active (shown at checkout)
          </label>
          {formError && <p className="text-sm text-accent">{formError}</p>}
          <div className="pt-2 flex items-center gap-4">
            <Button type="submit" disabled={busy}>
              {busy ? 'Saving…' : editId ? 'Save changes' : 'Create option'}
            </Button>
            {editId && (
              <button
                type="button"
                onClick={resetForm}
                className="text-sm text-ink-soft underline underline-offset-4 hover:text-ink"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
      </section>

      <section>
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6 flex items-baseline justify-between">
          <span>
            <span aria-hidden className="text-rule-strong mx-2">/</span>
            Existing options
          </span>
          <span className="tnum text-ink-faint">{loading ? '—' : `${options.length} total`}</span>
        </h2>

        {loading && <p className="text-sm text-ink-soft">Loading.</p>}
        {loadError && <p className="text-sm text-accent mb-4">{loadError}</p>}
        {!loading && options.length === 0 && !loadError && (
          <p className="text-sm text-ink-faint">No delivery options yet.</p>
        )}

        <ul className="divide-y divide-rule">
          {options.map((o) => (
            <li key={o.id} className="flex items-center justify-between gap-4 py-3">
              <div>
                <p className="text-ink">
                  {o.label}{' '}
                  {!o.active && (
                    <span className="text-xs text-ink-faint">(inactive)</span>
                  )}
                </p>
                <p className="text-xs text-ink-faint mt-0.5">
                  <span className="tnum">{o.code}</span>
                  {o.eta_label ? ` · ${o.eta_label}` : ''}
                </p>
              </div>
              <div className="flex items-center gap-4">
                <span className="tnum text-ink">{formatPrice(o.price_cents)}</span>
                <button
                  type="button"
                  onClick={() => startEdit(o)}
                  className="text-xs text-ink-soft underline underline-offset-4 hover:text-ink"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(o)}
                  disabled={rowBusyId === o.id}
                  className="text-xs text-accent underline underline-offset-4 disabled:opacity-50"
                >
                  {rowBusyId === o.id ? 'Deleting…' : 'Delete'}
                </button>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </Page>
  )
}
