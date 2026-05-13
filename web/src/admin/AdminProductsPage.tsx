import { useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import {
  api,
  ApiError,
  formatPrice,
  type ProductListItem,
  type Category,
  type Brand,
  type ProductDetail,
} from '../lib/api'

type EditState = {
  name: string
  priceDollars: string
  stockQuantity: string
}

type NewProductState = {
  name: string
  description: string
  priceDollars: string
  stockQuantity: string
  categoryId: string
  brandId: string
  weightG: string
  dimensionsCm: string
}

const emptyNewProduct: NewProductState = {
  name: '',
  description: '',
  priceDollars: '',
  stockQuantity: '',
  categoryId: '',
  brandId: '',
  weightG: '',
  dimensionsCm: '',
}

function flattenCategories(tree: Category[], depth = 0): Array<{ id: string; label: string }> {
  const out: Array<{ id: string; label: string }> = []
  for (const c of tree) {
    out.push({ id: c.id, label: `${'— '.repeat(depth)}${c.name}` })
    if (c.children?.length) out.push(...flattenCategories(c.children, depth + 1))
  }
  return out
}

export function AdminProductsPage() {
  const [products, setProducts] = useState<ProductListItem[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [edit, setEdit] = useState<EditState | null>(null)
  const [editBusy, setEditBusy] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [detail, setDetail] = useState<ProductDetail | null>(null)
  const [detailBusy, setDetailBusy] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)

  const [imageUrl, setImageUrl] = useState('')
  const [imagePrimary, setImagePrimary] = useState(false)
  const [imageBusy, setImageBusy] = useState(false)
  const [imageError, setImageError] = useState<string | null>(null)

  const [newProduct, setNewProduct] = useState<NewProductState>(emptyNewProduct)
  const [createBusy, setCreateBusy] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setLoadError(null)
      try {
        const [list, cats, brs] = await Promise.all([
          api.adminListProducts(),
          api.listCategories(),
          api.listBrands(),
        ])
        if (cancelled) return
        setProducts(list.products)
        setCategories(cats)
        setBrands(brs)
      } catch (err) {
        if (cancelled) return
        setLoadError(
          err instanceof ApiError ? err.message : 'Could not load products.',
        )
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [])

  async function loadDetail(id: string) {
    setDetailBusy(true)
    setDetailError(null)
    try {
      const d = await api.getProduct(id)
      setDetail(d)
    } catch (err) {
      setDetailError(
        err instanceof ApiError ? err.message : 'Could not load product detail.',
      )
    } finally {
      setDetailBusy(false)
    }
  }

  function toggleExpand(id: string) {
    if (expandedId === id) {
      setExpandedId(null)
      setDetail(null)
      setDetailError(null)
      setImageUrl('')
      setImagePrimary(false)
      setImageError(null)
      return
    }
    setExpandedId(id)
    setDetail(null)
    setImageUrl('')
    setImagePrimary(false)
    setImageError(null)
    loadDetail(id)
  }

  function beginEdit(p: ProductListItem) {
    setEditingId(p.id)
    setEditError(null)
    setEdit({
      name: p.name,
      priceDollars: (p.price / 100).toFixed(2),
      stockQuantity: String(p.stock_quantity),
    })
  }

  function cancelEdit() {
    setEditingId(null)
    setEdit(null)
    setEditError(null)
  }

  async function saveEdit(id: string) {
    if (!edit) return
    const priceCents = Math.round(parseFloat(edit.priceDollars) * 100)
    const stock = parseInt(edit.stockQuantity, 10)
    if (!Number.isFinite(priceCents) || priceCents < 0) {
      setEditError('Price must be a non-negative number.')
      return
    }
    if (!Number.isInteger(stock) || stock < 0) {
      setEditError('Stock must be a non-negative integer.')
      return
    }
    if (!edit.name.trim()) {
      setEditError('Name cannot be empty.')
      return
    }
    setEditBusy(true)
    setEditError(null)
    try {
      const updated = await api.adminUpdateProduct(id, {
        name: edit.name.trim(),
        price: priceCents,
        stock_quantity: stock,
      })
      setProducts((prev) =>
        prev.map((p) =>
          p.id === id
            ? {
                ...p,
                name: updated.name,
                price: updated.price,
                stock_quantity: updated.stock_quantity,
              }
            : p,
        ),
      )
      cancelEdit()
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : 'Save failed.')
    } finally {
      setEditBusy(false)
    }
  }

  async function deleteProduct(id: string, name: string) {
    if (!window.confirm(`Delete "${name}"? This cannot be undone.`)) return
    const snapshot = products
    setProducts((prev) => prev.filter((p) => p.id !== id))
    if (expandedId === id) {
      setExpandedId(null)
      setDetail(null)
    }
    try {
      await api.adminDeleteProduct(id)
    } catch (err) {
      setProducts(snapshot)
      setLoadError(err instanceof ApiError ? err.message : 'Delete failed.')
    }
  }

  async function addImage(productId: string) {
    if (!imageUrl.trim()) {
      setImageError('Image URL is required.')
      return
    }
    setImageBusy(true)
    setImageError(null)
    try {
      await api.adminAddProductImage(productId, imageUrl.trim(), imagePrimary)
      setImageUrl('')
      setImagePrimary(false)
      await loadDetail(productId)
    } catch (err) {
      setImageError(err instanceof ApiError ? err.message : 'Could not add image.')
    } finally {
      setImageBusy(false)
    }
  }

  async function deleteImage(productId: string, imageId: string) {
    if (!window.confirm('Delete this image?')) return
    try {
      await api.adminDeleteProductImage(productId, imageId)
      await loadDetail(productId)
    } catch (err) {
      setImageError(err instanceof ApiError ? err.message : 'Could not delete image.')
    }
  }

  async function createProduct(e: React.FormEvent) {
    e.preventDefault()
    const priceCents = Math.round(parseFloat(newProduct.priceDollars) * 100)
    const stock = parseInt(newProduct.stockQuantity, 10)
    const weight = parseInt(newProduct.weightG, 10)
    if (!newProduct.name.trim()) {
      setCreateError('Name is required.')
      return
    }
    if (!Number.isFinite(priceCents) || priceCents < 0) {
      setCreateError('Price must be a non-negative number.')
      return
    }
    if (!Number.isInteger(stock) || stock < 0) {
      setCreateError('Stock must be a non-negative integer.')
      return
    }
    if (!Number.isInteger(weight) || weight < 0) {
      setCreateError('Weight must be a non-negative integer (grams).')
      return
    }
    const body: Record<string, unknown> = {
      name: newProduct.name.trim(),
      price: priceCents,
      stock_quantity: stock,
      weight_g: weight,
    }
    if (newProduct.description.trim()) body.description = newProduct.description.trim()
    if (newProduct.categoryId) body.category_id = newProduct.categoryId
    if (newProduct.brandId) body.brand_id = newProduct.brandId
    if (newProduct.dimensionsCm.trim()) {
      const dim = parseFloat(newProduct.dimensionsCm)
      if (Number.isFinite(dim) && dim >= 0) body.dimensions_cm = dim
    }

    setCreateBusy(true)
    setCreateError(null)
    try {
      const created = await api.adminCreateProduct(body)
      const cat = categories
        .flatMap((c) => [c, ...(c.children ?? [])])
        .find((c) => c.id === created.category_id)
      const brand = brands.find((b) => b.id === created.brand_id)
      setProducts((prev) => [
        {
          id: created.id,
          name: created.name,
          price: created.price,
          stock_quantity: created.stock_quantity,
          category_name: cat?.name ?? null,
          brand_name: brand?.name ?? null,
          primary_image: null,
        },
        ...prev,
      ])
      setNewProduct(emptyNewProduct)
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : 'Could not create product.')
    } finally {
      setCreateBusy(false)
    }
  }

  const flatCategories = flattenCategories(categories)

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Catalogue"
        title="Products."
        caption="Create, edit, and remove the products that appear in the storefront."
      />

      <section className="max-w-3xl mb-16">
        <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
          <span className="tnum">01</span>
          <span aria-hidden className="text-rule-strong mx-2">
            /
          </span>
          New product
        </h2>
        <form onSubmit={createProduct} className="space-y-6">
          <Field
            label="Name"
            required
            value={newProduct.name}
            onChange={(e) =>
              setNewProduct((s) => ({ ...s, name: e.target.value }))
            }
          />
          <label className="block">
            <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
              Description
            </span>
            <textarea
              className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink placeholder-ink-faint transition-colors"
              style={{ borderRadius: 0, minHeight: '4rem' }}
              value={newProduct.description}
              onChange={(e) =>
                setNewProduct((s) => ({ ...s, description: e.target.value }))
              }
            />
          </label>
          <div className="grid grid-cols-2 gap-6">
            <Field
              label="Price (USD)"
              type="number"
              step="0.01"
              min="0"
              required
              value={newProduct.priceDollars}
              onChange={(e) =>
                setNewProduct((s) => ({ ...s, priceDollars: e.target.value }))
              }
            />
            <Field
              label="Stock"
              type="number"
              min="0"
              step="1"
              required
              value={newProduct.stockQuantity}
              onChange={(e) =>
                setNewProduct((s) => ({ ...s, stockQuantity: e.target.value }))
              }
            />
          </div>
          <div className="grid grid-cols-2 gap-6">
            <label className="block">
              <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
                Category
              </span>
              <select
                className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors"
                style={{ borderRadius: 0 }}
                value={newProduct.categoryId}
                onChange={(e) =>
                  setNewProduct((s) => ({ ...s, categoryId: e.target.value }))
                }
              >
                <option value="">— None —</option>
                {flatCategories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
                Brand
              </span>
              <select
                className="w-full bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-2 text-ink transition-colors"
                style={{ borderRadius: 0 }}
                value={newProduct.brandId}
                onChange={(e) =>
                  setNewProduct((s) => ({ ...s, brandId: e.target.value }))
                }
              >
                <option value="">— None —</option>
                {brands.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="grid grid-cols-2 gap-6">
            <Field
              label="Weight (g)"
              type="number"
              min="0"
              step="1"
              required
              value={newProduct.weightG}
              onChange={(e) =>
                setNewProduct((s) => ({ ...s, weightG: e.target.value }))
              }
            />
            <Field
              label="Dimension (cm)"
              type="number"
              min="0"
              step="0.1"
              value={newProduct.dimensionsCm}
              onChange={(e) =>
                setNewProduct((s) => ({ ...s, dimensionsCm: e.target.value }))
              }
            />
          </div>
          {createError && (
            <p className="text-sm text-accent">{createError}</p>
          )}
          <div className="pt-2">
            <Button type="submit" disabled={createBusy}>
              {createBusy ? 'Creating.' : 'Create product'}
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
            Catalogue
          </span>
          <span className="tnum text-ink-faint">
            {loading ? '—' : `${products.length} products`}
          </span>
        </h2>

        {loadError && (
          <p className="text-sm text-accent mb-4">{loadError}</p>
        )}
        {loading && <p className="text-sm text-ink-soft">Loading.</p>}

        <ul className="divide-y divide-rule">
          {products.map((p) => {
            const isEditing = editingId === p.id
            const isExpanded = expandedId === p.id
            const lowStock = p.stock_quantity <= 5
            return (
              <li key={p.id} className="py-4">
                <div className="grid grid-cols-[1fr_8rem_5rem_8rem_auto] items-baseline gap-4">
                  {isEditing && edit ? (
                    <>
                      <input
                        className="bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-1 text-ink"
                        style={{ borderRadius: 0 }}
                        value={edit.name}
                        onChange={(e) =>
                          setEdit((s) => (s ? { ...s, name: e.target.value } : s))
                        }
                      />
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        className="bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-1 text-ink tnum text-right"
                        style={{ borderRadius: 0 }}
                        value={edit.priceDollars}
                        onChange={(e) =>
                          setEdit((s) =>
                            s ? { ...s, priceDollars: e.target.value } : s,
                          )
                        }
                      />
                      <input
                        type="number"
                        min="0"
                        step="1"
                        className="bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-1 text-ink tnum text-right"
                        style={{ borderRadius: 0 }}
                        value={edit.stockQuantity}
                        onChange={(e) =>
                          setEdit((s) =>
                            s ? { ...s, stockQuantity: e.target.value } : s,
                          )
                        }
                      />
                      <div className="text-xs text-ink-faint">
                        {p.category_name ?? '—'}
                        {p.brand_name ? ` · ${p.brand_name}` : ''}
                      </div>
                      <div className="flex items-center gap-3 justify-end">
                        <Button
                          onClick={() => saveEdit(p.id)}
                          disabled={editBusy}
                        >
                          {editBusy ? 'Saving.' : 'Save'}
                        </Button>
                        <Button variant="link" onClick={cancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    </>
                  ) : (
                    <>
                      <button
                        type="button"
                        className="text-left text-ink hover:text-accent transition-colors cursor-pointer"
                        onClick={() => beginEdit(p)}
                      >
                        {p.name}
                      </button>
                      <span className="tnum text-right text-ink">
                        {formatPrice(p.price)}
                      </span>
                      <span
                        className={`tnum text-right ${
                          lowStock ? 'text-accent' : 'text-ink-soft'
                        }`}
                      >
                        {p.stock_quantity}
                      </span>
                      <span className="text-xs text-ink-faint">
                        {p.category_name ?? '—'}
                        {p.brand_name ? ` · ${p.brand_name}` : ''}
                      </span>
                      <div className="flex items-center gap-3 justify-end">
                        <button
                          type="button"
                          className="text-sm text-ink-soft hover:text-ink underline underline-offset-4 cursor-pointer"
                          onClick={() => toggleExpand(p.id)}
                          aria-expanded={isExpanded}
                        >
                          {isExpanded ? 'Hide images' : 'Images'}
                        </button>
                        <button
                          type="button"
                          className="text-sm text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer"
                          onClick={() => deleteProduct(p.id, p.name)}
                        >
                          Delete
                        </button>
                      </div>
                    </>
                  )}
                </div>

                {isEditing && editError && (
                  <p className="text-sm text-accent mt-2">{editError}</p>
                )}

                {isExpanded && (
                  <div className="mt-6 pl-0">
                    {detailBusy && (
                      <p className="text-sm text-ink-soft">Loading.</p>
                    )}
                    {detailError && (
                      <p className="text-sm text-accent">{detailError}</p>
                    )}
                    {detail && detail.id === p.id && (
                      <div className="space-y-6">
                        {detail.images.length === 0 ? (
                          <p className="text-sm text-ink-faint">
                            No images yet.
                          </p>
                        ) : (
                          <ul className="flex flex-wrap gap-4">
                            {detail.images.map((img) => (
                              <li
                                key={img.id}
                                className="flex flex-col gap-2 w-32"
                              >
                                <div className="aspect-square bg-sunken overflow-hidden">
                                  <img
                                    src={img.url}
                                    alt=""
                                    loading="lazy"
                                    className="w-full h-full object-cover"
                                  />
                                </div>
                                <div className="flex items-baseline justify-between text-xs">
                                  {img.is_primary ? (
                                    <span className="uc-tight text-accent">
                                      Primary
                                    </span>
                                  ) : (
                                    <span className="text-ink-faint">—</span>
                                  )}
                                  <button
                                    type="button"
                                    className="text-ink-soft hover:text-accent underline underline-offset-4 cursor-pointer"
                                    onClick={() => deleteImage(p.id, img.id)}
                                  >
                                    Delete
                                  </button>
                                </div>
                              </li>
                            ))}
                          </ul>
                        )}

                        <div className="border-t border-rule pt-4">
                          <div className="uc-tight text-[0.7rem] text-ink-faint mb-3">
                            Add image
                          </div>
                          <div className="flex items-end gap-4 flex-wrap">
                            <div className="flex-1 min-w-[18rem]">
                              <Field
                                label="URL"
                                type="url"
                                placeholder="https://…"
                                value={imageUrl}
                                onChange={(e) => setImageUrl(e.target.value)}
                              />
                            </div>
                            <label className="flex items-baseline gap-2 text-sm text-ink-soft pb-2">
                              <input
                                type="checkbox"
                                checked={imagePrimary}
                                onChange={(e) =>
                                  setImagePrimary(e.target.checked)
                                }
                              />
                              Mark primary
                            </label>
                            <Button
                              onClick={() => addImage(p.id)}
                              disabled={imageBusy}
                            >
                              {imageBusy ? 'Adding.' : 'Add image'}
                            </Button>
                          </div>
                          {imageError && (
                            <p className="text-sm text-accent mt-3">
                              {imageError}
                            </p>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </li>
            )
          })}
        </ul>

        {!loading && products.length === 0 && !loadError && (
          <p className="text-sm text-ink-faint mt-4">
            No products yet. Use the form above to add one.
          </p>
        )}
      </section>
    </Page>
  )
}
