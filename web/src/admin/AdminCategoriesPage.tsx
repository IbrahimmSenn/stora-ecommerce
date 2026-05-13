import { useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { api, ApiError, type Category } from '../lib/api'

function flatten(tree: Category[], depth = 0): Array<{ id: string; label: string }> {
  const out: Array<{ id: string; label: string }> = []
  for (const c of tree) {
    out.push({ id: c.id, label: `${'— '.repeat(depth)}${c.name}` })
    if (c.children?.length) out.push(...flatten(c.children, depth + 1))
  }
  return out
}

function CategoryNode({ node, depth }: { node: Category; depth: number }) {
  return (
    <li>
      <div
        className="flex items-baseline justify-between gap-4 py-2"
        style={{ paddingLeft: `${depth * 1.5}rem` }}
      >
        <span className="text-ink">{node.name}</span>
        <span className="tnum text-xs text-ink-faint">{node.slug}</span>
      </div>
      {node.children && node.children.length > 0 && (
        <ul>
          {node.children.map((c) => (
            <CategoryNode key={c.id} node={c} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  )
}

export function AdminCategoriesPage() {
  const [tree, setTree] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [parentId, setParentId] = useState('')
  const [busy, setBusy] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  async function refresh() {
    setLoading(true)
    setLoadError(null)
    try {
      const t = await api.listCategories()
      setTree(t)
    } catch (err) {
      setLoadError(
        err instanceof ApiError ? err.message : 'Could not load categories.',
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
    if (!name.trim() || !slug.trim()) {
      setCreateError('Name and slug are required.')
      return
    }
    setBusy(true)
    setCreateError(null)
    try {
      await api.adminCreateCategory({
        name: name.trim(),
        slug: slug.trim(),
        parent_id: parentId || undefined,
      })
      setName('')
      setSlug('')
      setParentId('')
      await refresh()
    } catch (err) {
      setCreateError(
        err instanceof ApiError ? err.message : 'Could not create category.',
      )
    } finally {
      setBusy(false)
    }
  }

  const flat = flatten(tree)

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Catalogue"
        title="Categories."
        caption="The taxonomy products are organised under."
      />

      <section className="max-w-2xl mb-16">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
          <span className="tnum">01</span>
          <span aria-hidden className="text-rule-strong mx-2">
            /
          </span>
          New category
        </h2>
        <form onSubmit={handleCreate} className="space-y-6">
          <Field
            label="Name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Field
            label="Slug"
            required
            placeholder="e.g. herbs, vegetables, lavender"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            hint="Lowercase, hyphen-separated. Used in URLs."
          />
          <label className="block">
            <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
              Parent
            </span>
            <select
              className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors"
              style={{ borderRadius: 0 }}
              value={parentId}
              onChange={(e) => setParentId(e.target.value)}
            >
              <option value="">— None (top-level) —</option>
              {flat.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
            </select>
          </label>
          {createError && <p className="text-sm text-accent">{createError}</p>}
          <div className="pt-2">
            <Button type="submit" disabled={busy}>
              {busy ? 'Creating.' : 'Create category'}
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
            Existing categories
          </span>
          <span className="tnum text-ink-faint">
            {loading ? '—' : `${flat.length} total`}
          </span>
        </h2>

        {loading && <p className="text-sm text-ink-soft">Loading.</p>}
        {loadError && <p className="text-sm text-accent">{loadError}</p>}
        {!loading && tree.length === 0 && !loadError && (
          <p className="text-sm text-ink-faint">No categories yet.</p>
        )}

        <ul className="divide-y divide-rule">
          {tree.map((c) => (
            <CategoryNode key={c.id} node={c} depth={0} />
          ))}
        </ul>

        <p className="text-xs text-ink-faint mt-8">
          Categories cannot be edited or deleted from the admin UI.
        </p>
      </section>
    </Page>
  )
}
