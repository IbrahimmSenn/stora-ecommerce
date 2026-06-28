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

// The tree response carries no parent_id, so derive parent and descendant sets
// from the nesting: parentOf for prefilling the edit form, descendants to keep a
// category from being reparented under itself (which would orphan its subtree).
function parentMap(tree: Category[], parent = '', acc: Record<string, string> = {}) {
  for (const c of tree) {
    acc[c.id] = parent
    if (c.children?.length) parentMap(c.children, c.id, acc)
  }
  return acc
}

function descendantsOf(tree: Category[], id: string): Set<string> {
  const find = (nodes: Category[]): Category | null => {
    for (const n of nodes) {
      if (n.id === id) return n
      const f = n.children ? find(n.children) : null
      if (f) return f
    }
    return null
  }
  const collect = (n: Category, acc: Set<string>) => {
    for (const c of n.children ?? []) {
      acc.add(c.id)
      collect(c, acc)
    }
  }
  const node = find(tree)
  const out = new Set<string>()
  if (node) collect(node, out)
  return out
}

function CategoryNode({
  node,
  depth,
  busyId,
  onEdit,
  onDelete,
}: {
  node: Category
  depth: number
  busyId: string | null
  onEdit: (node: Category) => void
  onDelete: (node: Category) => void
}) {
  return (
    <li>
      <div
        className="flex items-baseline justify-between gap-4 py-2"
        style={{ paddingLeft: `${depth * 1.5}rem` }}
      >
        <span className="text-ink">{node.name}</span>
        <span className="flex items-baseline gap-4">
          <span className="tnum text-xs text-ink-faint">{node.slug}</span>
          <button
            type="button"
            onClick={() => onEdit(node)}
            className="text-xs text-ink-soft underline underline-offset-4 hover:text-ink"
          >
            Edit
          </button>
          <button
            type="button"
            onClick={() => onDelete(node)}
            disabled={busyId === node.id}
            className="text-xs text-accent underline underline-offset-4 hover:text-accent disabled:opacity-50"
          >
            {busyId === node.id ? 'Deleting…' : 'Delete'}
          </button>
        </span>
      </div>
      {node.children && node.children.length > 0 && (
        <ul>
          {node.children.map((c) => (
            <CategoryNode
              key={c.id}
              node={c}
              depth={depth + 1}
              busyId={busyId}
              onEdit={onEdit}
              onDelete={onDelete}
            />
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

  const [editId, setEditId] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [parentId, setParentId] = useState('')
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [deleteBusyId, setDeleteBusyId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

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
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh()
  }, [])

  function resetForm() {
    setEditId(null)
    setName('')
    setSlug('')
    setParentId('')
    setFormError(null)
  }

  function startEdit(node: Category) {
    setEditId(node.id)
    setName(node.name)
    setSlug(node.slug)
    setParentId(parentMap(tree)[node.id] ?? '')
    setFormError(null)
    setDeleteError(null)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !slug.trim()) {
      setFormError('Name and slug are required.')
      return
    }
    setBusy(true)
    setFormError(null)
    try {
      const body = {
        name: name.trim(),
        slug: slug.trim(),
        parent_id: parentId || undefined,
      }
      if (editId) {
        await api.adminUpdateCategory(editId, body)
      } else {
        await api.adminCreateCategory(body)
      }
      resetForm()
      await refresh()
    } catch (err) {
      setFormError(
        err instanceof ApiError
          ? err.message
          : `Could not ${editId ? 'update' : 'create'} category.`,
      )
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete(node: Category) {
    if (!window.confirm(`Delete category “${node.name}”? This cannot be undone.`)) return
    setDeleteBusyId(node.id)
    setDeleteError(null)
    try {
      await api.adminDeleteCategory(node.id)
      if (editId === node.id) resetForm()
      await refresh()
    } catch (err) {
      setDeleteError(
        err instanceof ApiError ? err.message : 'Could not delete category.',
      )
    } finally {
      setDeleteBusyId(null)
    }
  }

  // Parent options: for the edit form, exclude the edited category and its
  // descendants so it can't be reparented into its own subtree.
  const excluded = editId ? new Set([editId, ...descendantsOf(tree, editId)]) : new Set<string>()
  const parentOptions = flatten(tree).filter((c) => !excluded.has(c.id))
  const totalCount = flatten(tree).length

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Catalogue"
        title="Categories."
        caption="The taxonomy products are organised under."
      />

      <section className="max-w-2xl mb-16">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
          <span aria-hidden className="text-rule-strong mx-2">
            /
          </span>
          {editId ? 'Edit category' : 'New category'}
        </h2>
        <form onSubmit={handleSubmit} className="space-y-6">
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
              {parentOptions.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
            </select>
          </label>
          {formError && <p className="text-sm text-accent">{formError}</p>}
          <div className="pt-2 flex items-center gap-4">
            <Button type="submit" disabled={busy}>
              {busy
                ? editId
                  ? 'Saving…'
                  : 'Creating…'
                : editId
                  ? 'Save changes'
                  : 'Create category'}
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
            <span aria-hidden className="text-rule-strong mx-2">
              /
            </span>
            Existing categories
          </span>
          <span className="tnum text-ink-faint">
            {loading ? '—' : `${totalCount} total`}
          </span>
        </h2>

        {loading && <p className="text-sm text-ink-soft">Loading.</p>}
        {loadError && <p className="text-sm text-accent">{loadError}</p>}
        {deleteError && <p className="text-sm text-accent mb-4">{deleteError}</p>}
        {!loading && tree.length === 0 && !loadError && (
          <p className="text-sm text-ink-faint">No categories yet.</p>
        )}

        <ul className="divide-y divide-rule">
          {tree.map((c) => (
            <CategoryNode
              key={c.id}
              node={c}
              depth={0}
              busyId={deleteBusyId}
              onEdit={startEdit}
              onDelete={handleDelete}
            />
          ))}
        </ul>
      </section>
    </Page>
  )
}
