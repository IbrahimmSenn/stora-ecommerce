// gen-seed.mjs — generate a realistic product catalogue from the DummyJSON
// open dataset (https://dummyjson.com). Emits migrations/seed.sql (brands,
// categories, products, product_images, sale_price updates), downloads product
// images into web/public/products/, and writes internal/seed/reviews_gen.go so
// the app's demo reviews reference the generated product IDs.
//
// Run: node scripts/gen-seed.mjs
// Deterministic: product/category/brand UUIDs are derived from stable keys, so
// re-running produces the same IDs (and the same image filenames).

import crypto from 'node:crypto'
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const IMG_DIR = path.join(ROOT, 'web/public/products')
const SEED_SQL = path.join(ROOT, 'migrations/seed.sql')
const REVIEWS_GO = path.join(ROOT, 'internal/seed/reviews_gen.go')

// Group the ~24 dataset categories into clean top-level storefront categories.
const GROUPS = [
  { name: 'Electronics', slug: 'electronics', cats: ['laptops', 'smartphones', 'tablets', 'mobile-accessories'] },
  { name: 'Beauty', slug: 'beauty', cats: ['beauty', 'fragrances', 'skin-care'] },
  { name: 'Home', slug: 'home', cats: ['home-decoration', 'kitchen-accessories'] },
  { name: 'Furniture', slug: 'furniture', cats: ['furniture'] },
  { name: 'Clothing', slug: 'clothing', cats: ['mens-shirts', 'tops', 'womens-dresses'] },
  { name: 'Shoes', slug: 'shoes', cats: ['mens-shoes', 'womens-shoes'] },
  { name: 'Accessories', slug: 'accessories', cats: ['mens-watches', 'womens-watches', 'sunglasses', 'womens-bags', 'womens-jewellery'] },
  { name: 'Sports', slug: 'sports', cats: ['sports-accessories'] },
  { name: 'Groceries', slug: 'groceries', cats: ['groceries'] },
  { name: 'Automotive', slug: 'automotive', cats: ['motorcycle', 'vehicle'] },
]
const catToGroup = new Map()
for (const g of GROUPS) for (const c of g.cats) catToGroup.set(c, g)

// Demo user IDs that internal/seed/seed.go creates (reviews must reference them).
const DEMO_USERS = [
  'c0000000-0000-0000-0000-000000000001', // customer@shop.com
  'c0000000-0000-0000-0000-000000000003', // test3@test.com
  'a0000000-0000-0000-0000-000000000001', // admin@shop.com
]

function uuid(name) {
  const h = crypto.createHash('sha1').update(name).digest()
  const b = Buffer.from(h.subarray(0, 16))
  b[6] = (b[6] & 0x0f) | 0x50 // version 5
  b[8] = (b[8] & 0x3f) | 0x80 // RFC-4122 variant
  const x = b.toString('hex')
  return `${x.slice(0, 8)}-${x.slice(8, 12)}-${x.slice(12, 16)}-${x.slice(16, 20)}-${x.slice(20, 32)}`
}

const sqlStr = (s) => `'${String(s ?? '').replace(/'/g, "''")}'`
const goStr = (s) =>
  `"${String(s ?? '').replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/[\r\n]+/g, ' ').trim()}"`

async function pool(items, n, worker) {
  const results = []
  let i = 0
  await Promise.all(
    Array.from({ length: n }, async () => {
      while (i < items.length) {
        const idx = i++
        results[idx] = await worker(items[idx], idx)
      }
    }),
  )
  return results
}

async function main() {
  console.log('Fetching DummyJSON products…')
  const res = await fetch('https://dummyjson.com/products?limit=0')
  if (!res.ok) throw new Error(`dataset fetch failed: ${res.status}`)
  const all = (await res.json()).products
  const products = all.filter((p) => catToGroup.has(p.category))
  console.log(`Got ${all.length} products, keeping ${products.length} in mapped categories.`)

  for (const g of GROUPS) g.id = uuid('category:' + g.slug)

  // Reset image dir.
  await fs.rm(IMG_DIR, { recursive: true, force: true })
  await fs.mkdir(IMG_DIR, { recursive: true })

  // Brands present in the kept set.
  const brands = new Map()
  for (const p of products) {
    const name = (p.brand || '').trim()
    if (name && !brands.has(name)) brands.set(name, uuid('brand:' + name))
  }

  // Build product rows + download images.
  const rows = []
  const imageRows = []
  const saleRows = []
  const reviewRows = []

  const downloads = []
  products.forEach((p, idx) => {
    const id = uuid('product:' + p.id)
    const g = catToGroup.get(p.category)
    const priceC = Math.round(p.price * 100)
    const brandId = p.brand ? brands.get(p.brand.trim()) : null
    const weightG = p.weight ? Math.round(p.weight) : null
    const dims = p.dimensions
      ? Math.round(Math.max(p.dimensions.width, p.dimensions.height, p.dimensions.depth) * 100) / 100
      : null

    rows.push({ id, name: p.title, description: p.description, priceC, stock: p.stock, catId: g.id, brandId, weightG, dims })

    // Sale on a ~1/3 subset with a meaningful discount, kept under the price.
    if (p.discountPercentage >= 10 && p.id % 3 === 0) {
      const saleC = Math.round(p.price * (1 - p.discountPercentage / 100) * 100)
      if (saleC > 0 && saleC < priceC) saleRows.push({ id, saleC })
    }

    const imgs = [...new Set(p.images || [])].slice(0, 3)
    imgs.forEach((url, n) => {
      const ext = (url.match(/\.(webp|jpe?g|png|gif)(\?|$)/i)?.[1] || 'jpg').toLowerCase()
      const file = `${id}-${n + 1}.${ext}`
      downloads.push({ url, file })
      imageRows.push({ productId: id, url: `/products/${file}`, isPrimary: n === 0 })
    })

    // Demo reviews: spread across categories, real DummyJSON comments mapped to
    // our demo users (max one per user per product per the unique constraint).
    if (idx % 6 === 0 && Array.isArray(p.reviews)) {
      p.reviews.slice(0, 3).forEach((r, n) => {
        reviewRows.push({ user: DEMO_USERS[n], productId: id, comment: r.comment, rating: r.rating })
      })
    }
  })

  console.log(`Downloading ${downloads.length} images…`)
  let ok = 0
  await pool(downloads, 8, async (d) => {
    try {
      const r = await fetch(d.url)
      if (!r.ok) throw new Error(String(r.status))
      await fs.writeFile(path.join(IMG_DIR, d.file), Buffer.from(await r.arrayBuffer()))
      ok++
    } catch (e) {
      console.warn(`  skip ${d.url}: ${e.message}`)
    }
  })
  console.log(`Downloaded ${ok}/${downloads.length} images.`)

  // Emit seed.sql
  const out = []
  out.push('-- Seed data — realistic catalogue generated from DummyJSON (scripts/gen-seed.mjs).')
  out.push('-- Dev logins: admin@shop.com / admin123 (ADMIN_PASSWORD overrides it in demo deployments) · customer@shop.com / customer123 · test3@test.com / test123')
  out.push('--')
  out.push('-- NOTE: demo users + their reviews are seeded by the application at startup')
  out.push('-- (internal/seed) because user email is AES-GCM encrypted at rest. This file')
  out.push('-- seeds brands/categories/products/images only.')
  out.push('')
  out.push('INSERT INTO brands (id, name) VALUES')
  out.push([...brands].map(([name, id]) => `  (${sqlStr(id)}, ${sqlStr(name)})`).join(',\n'))
  out.push('ON CONFLICT (name) DO NOTHING;')
  out.push('')
  out.push('INSERT INTO categories (id, name, slug, parent_id) VALUES')
  out.push(GROUPS.map((g) => `  (${sqlStr(g.id)}, ${sqlStr(g.name)}, ${sqlStr(g.slug)}, NULL)`).join(',\n'))
  out.push('ON CONFLICT (name) DO NOTHING;')
  out.push('')
  out.push('INSERT INTO products (id, name, description, price, stock_quantity, category_id, brand_id, weight_g, dimensions_cm) VALUES')
  out.push(
    rows
      .map(
        (r) =>
          `  (${sqlStr(r.id)}, ${sqlStr(r.name)}, ${sqlStr(r.description)}, ${r.priceC}, ${r.stock}, ${sqlStr(r.catId)}, ${r.brandId ? sqlStr(r.brandId) : 'NULL'}, ${r.weightG ?? 'NULL'}, ${r.dims ?? 'NULL'})`,
      )
      .join(',\n'),
  )
  out.push('ON CONFLICT (id) DO NOTHING;')
  out.push('')
  if (saleRows.length) {
    out.push('-- Sale prices so the storefront shows real discounts.')
    out.push('UPDATE products SET sale_price = v.sale_price FROM (VALUES')
    out.push(saleRows.map((s) => `  (${sqlStr(s.id)}::uuid, ${s.saleC}::bigint)`).join(',\n'))
    out.push(') AS v(id, sale_price) WHERE products.id = v.id;')
    out.push('')
  }
  out.push('INSERT INTO product_images (product_id, url, is_primary) VALUES')
  out.push(imageRows.map((i) => `  (${sqlStr(i.productId)}, ${sqlStr(i.url)}, ${i.isPrimary})`).join(',\n'))
  out.push('ON CONFLICT (product_id, url) DO NOTHING;')
  out.push('')
  await fs.writeFile(SEED_SQL, out.join('\n'))
  console.log(`Wrote ${SEED_SQL} (${rows.length} products, ${brands.size} brands, ${saleRows.length} on sale, ${imageRows.length} images).`)

  // Emit reviews_gen.go
  const go = []
  go.push('// Code generated by scripts/gen-seed.mjs. DO NOT EDIT.')
  go.push('')
  go.push('package seed')
  go.push('')
  go.push('var demoReviews = []demoReview{')
  for (const r of reviewRows) {
    go.push(`\t{${goStr(r.user)}, ${goStr(r.productId)}, ${goStr(r.comment)}, ${r.rating}},`)
  }
  go.push('}')
  go.push('')
  await fs.writeFile(REVIEWS_GO, go.join('\n'))
  console.log(`Wrote ${REVIEWS_GO} (${reviewRows.length} demo reviews).`)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
