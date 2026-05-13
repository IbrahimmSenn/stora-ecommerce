import { useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError, type Brand } from '../lib/api'

export function AdminBrandsPage() {
  const [brands, setBrands] = useState<Brand[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  async function refresh() {
    setLoading(true)
    setLoadError(null)
    try {
      const list = await api.listBrands()
      setBrands(list)
    } catch (err) {
      setLoadError(
        err instanceof ApiError ? err.message : 'Could not load brands.',
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) {
      setCreateError('Name is required.')
      return
    }
    setBusy(true)
    setCreateError(null)
    try {
      const created = await api.adminCreateBrand({ name: name.trim() })
      setBrands((prev) => [created, ...prev])
      setName('')
    } catch (err) {
      setCreateError(
        err instanceof ApiError ? err.message : 'Could not create brand.',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Catalogue"
        title="Brands."
        caption="The makers and labels behind the products."
      />

      <section className="max-w-xl mb-16">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
          <span className="tnum">01</span>
          <span aria-hidden className="text-rule-strong mx-2">
            /
          </span>
          New brand
        </h2>
        <form onSubmit={handleCreate} className="space-y-6">
          <Field
            label="Name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          {createError && <p className="text-sm text-accent">{createError}</p>}
          <div className="pt-2">
            <Button type="submit" disabled={busy}>
              {busy ? 'Creating.' : 'Create brand'}
            </Button>
          </div>
        </form>
      </section>

      <section>
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6 flex items-baseline justify-between">
          <span>
            <span className="tnum">02</span>
            <span aria-hidden className="text-rule-strong mx-2">
              /
            </span>
            Existing brands
          </span>
          <span className="tnum text-ink-faint">
            {loading ? '—' : `${brands.length} total`}
          </span>
        </h2>

        {loading && <p className="text-sm text-ink-soft">Loading.</p>}
        {loadError && <p className="text-sm text-accent">{loadError}</p>}
        {!loading && brands.length === 0 && !loadError && (
          <p className="text-sm text-ink-faint">No brands yet.</p>
        )}

        <ul className="divide-y divide-rule">
          {brands.map((b) => (
            <li
              key={b.id}
              className="flex items-baseline justify-between py-3"
            >
              <span className="text-ink">{b.name}</span>
              <span className="tnum text-xs text-ink-faint">{b.id}</span>
            </li>
          ))}
        </ul>

        <p className="text-xs text-ink-faint mt-8">
          Brands cannot be edited or deleted from the admin UI.
        </p>
      </section>
    </Page>
  )
}
