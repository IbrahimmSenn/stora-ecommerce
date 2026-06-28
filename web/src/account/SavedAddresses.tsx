import { useEffect, useState } from 'react'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError, type SavedAddress, type SavedAddressInput } from '../lib/api'

const empty: SavedAddressInput = {
  label: '',
  recipient_name: '',
  line1: '',
  line2: '',
  city: '',
  region: '',
  postal_code: '',
  country: '',
  is_default: false,
}

export function SavedAddresses() {
  const [addresses, setAddresses] = useState<SavedAddress[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState<string | 'new' | null>(null)
  const [form, setForm] = useState<SavedAddressInput>(empty)
  const [busy, setBusy] = useState(false)

  function load() {
    setLoading(true)
    api
      .listAddresses()
      .then((r) => setAddresses(r.addresses))
      .catch((e) => setError(e instanceof ApiError ? e.message : 'Could not load addresses.'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load()
  }, [])

  function startAdd() {
    setForm(empty)
    setEditing('new')
    setError(null)
  }

  function startEdit(a: SavedAddress) {
    setForm({
      label: a.label ?? '',
      recipient_name: a.recipient_name,
      line1: a.line1,
      line2: a.line2 ?? '',
      city: a.city,
      region: a.region,
      postal_code: a.postal_code,
      country: a.country,
      is_default: a.is_default,
    })
    setEditing(a.id)
    setError(null)
  }

  async function save(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      if (editing === 'new') await api.createAddress(form)
      else if (editing) await api.updateAddress(editing, form)
      setEditing(null)
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save address.')
    } finally {
      setBusy(false)
    }
  }

  async function remove(id: string) {
    if (!window.confirm('Remove this address?')) return
    try {
      await api.deleteAddress(id)
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not remove address.')
    }
  }

  async function makeDefault(id: string) {
    try {
      await api.setDefaultAddress(id)
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not set default.')
    }
  }

  const set = (k: keyof SavedAddressInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }))

  return (
    <div className="space-y-5">
      <h2 className="font-display text-xl text-ink font-bold">Saved addresses</h2>
      <p className="text-sm text-ink-soft max-w-[55ch]">
        Stored encrypted and reused at checkout. Your default is filled in first.
      </p>

      {error && <p className="text-sm text-accent" role="alert">{error}</p>}

      {loading ? (
        <p className="text-sm text-ink-soft">Loading.</p>
      ) : addresses.length === 0 && editing === null ? (
        <p className="text-sm text-ink-faint">No saved addresses yet.</p>
      ) : (
        <ul className="flex flex-col divide-y divide-rule">
          {addresses.map((a) => (
            <li key={a.id} className="py-4 flex items-start justify-between gap-4">
              <div className="text-sm">
                <p className="text-ink">
                  {a.recipient_name}
                  {a.label && <span className="uc-tight text-[0.65rem] text-ink-faint ml-2">{a.label}</span>}
                  {a.is_default && <span className="uc-tight text-[0.65rem] text-accent ml-2">default</span>}
                </p>
                <p className="text-ink-soft">
                  {a.line1}{a.line2 ? `, ${a.line2}` : ''}, {a.city}, {a.region} {a.postal_code}, {a.country}
                </p>
              </div>
              <div className="flex flex-col items-end gap-1 shrink-0 text-xs">
                <button type="button" onClick={() => startEdit(a)} className="text-ink-soft hover:text-ink underline underline-offset-4 cursor-pointer">Edit</button>
                {!a.is_default && (
                  <button type="button" onClick={() => makeDefault(a.id)} className="text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer">Set default</button>
                )}
                <button type="button" onClick={() => remove(a.id)} className="text-negative hover:opacity-80 underline underline-offset-4 cursor-pointer">Remove</button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {editing !== null ? (
        <form onSubmit={save} className="border-t border-rule pt-5 grid grid-cols-1 sm:grid-cols-2 gap-4 max-w-xl">
          <Field label="Label (optional)" value={form.label ?? ''} onChange={set('label')} placeholder="Home, Work…" />
          <Field label="Recipient" required value={form.recipient_name} onChange={set('recipient_name')} />
          <Field label="Address line 1" required value={form.line1} onChange={set('line1')} />
          <Field label="Address line 2" value={form.line2 ?? ''} onChange={set('line2')} />
          <Field label="City" required value={form.city} onChange={set('city')} />
          <Field label="Region / State" required value={form.region} onChange={set('region')} />
          <Field label="Postal code" required value={form.postal_code} onChange={set('postal_code')} />
          <Field label="Country (2-letter)" required value={form.country} onChange={set('country')} placeholder="US" />
          <label className="flex items-center gap-2 text-sm text-ink-soft sm:col-span-2">
            <input type="checkbox" checked={!!form.is_default} onChange={(e) => setForm((f) => ({ ...f, is_default: e.target.checked }))} />
            Make this my default address
          </label>
          <div className="flex gap-3 sm:col-span-2">
            <Button type="submit" disabled={busy}>{busy ? 'Saving.' : 'Save address'}</Button>
            <button type="button" onClick={() => setEditing(null)} className="text-sm text-ink-faint hover:text-ink cursor-pointer">Cancel</button>
          </div>
        </form>
      ) : (
        <Button type="button" onClick={startAdd}>Add address</Button>
      )}
    </div>
  )
}
