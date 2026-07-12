// Minimal fetch wrapper. Sends cookies (for guest_session_id) and attaches
// the in-memory access token when one is set by AuthContext.

let accessToken: string | null = null

export function setAccessToken(t: string | null) {
  accessToken = t
}

export class ApiError extends Error {
  status: number
  /** Optional stable error code from the server. Frontends should branch on
   *  this rather than string-matching `message`. */
  code?: string
  constructor(status: number, message: string, code?: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

type Options = {
  method?: string
  body?: unknown
  /** Abort the request after this many ms. Default 20s. */
  timeoutMs?: number
}

export async function request<T>(path: string, opts: Options = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  const controller = new AbortController()
  const timeoutMs = opts.timeoutMs ?? 20_000
  const timer = window.setTimeout(() => controller.abort(), timeoutMs)

  let res: Response
  try {
    res = await fetch(path, {
      method: opts.method ?? 'GET',
      credentials: 'include',
      headers,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      signal: controller.signal,
    })
  } finally {
    window.clearTimeout(timer)
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()
  const data: unknown = text ? JSON.parse(text) : null

  if (!res.ok) {
    const obj =
      data && typeof data === 'object' ? (data as Record<string, unknown>) : null
    const msg =
      obj && typeof obj.error === 'string'
        ? (obj.error as string)
        : `request failed (${res.status})`
    const code = obj && typeof obj.code === 'string' ? (obj.code as string) : undefined
    throw new ApiError(res.status, msg, code)
  }
  return data as T
}

// --- Typed endpoints ---

export type ProductListItem = {
  id: string
  name: string
  price: number
  sale_price?: number | null
  stock_quantity: number
  category_name?: string | null
  brand_name?: string | null
  primary_image?: string | null
  avg_rating: number
  review_count: number
}

export type PublicReview = {
  id: string
  rating: number
  comment?: string | null
  helpful_count: number
  voted_by_me: boolean
  mine_to_edit: boolean
  created_at: string
}

export type ReviewListResult = {
  reviews: PublicReview[]
  total: number
  page: number
  page_size: number
  avg_rating: number
  distribution: Record<string, number>
}

export type ReviewSort = 'helpful' | 'newest' | 'highest' | 'lowest'

export type ReviewEligibility = {
  can_review: boolean
  has_purchased: boolean
  already_reviewed: boolean
  existing_rating?: number | null
  existing_pending: boolean
}

export type ProductsResponse = {
  products: ProductListItem[]
  total: number
  page: number
  page_size: number
}

export type ProductSuggestion = {
  id: string
  name: string
}

export type CartItem = {
  id: string
  product_id: string
  product_name: string
  product_price: number
  image_url?: string | null
  quantity: number
  stock: number
}

export type Cart = {
  id: string
  items: CartItem[]
  total: number
}

export type MergeStatus = {
  conflict: boolean
  auto_merged?: boolean
  guest_cart?: Cart
  user_cart?: Cart
}

export type LoginResponse = {
  access_token: string
  refresh_token: string
  expires_at: string
  token_type: string
}

export type RegisterRequest = {
  email: string
  password: string
  captcha_token?: string
}

export type TwoFactorSetupResponse = {
  secret: string
  /** base64-encoded PNG without the data: prefix. */
  qr_code: string
  recovery_codes: string[]
}

export type AdminProduct = {
  id: string
  name: string
  description?: string | null
  price: number
  sale_price?: number | null
  stock_quantity: number
  category_id?: string | null
  brand_id?: string | null
  weight_g?: number | null
  dimensions_cm?: number | null
  category_name?: string | null
  brand_name?: string | null
  created_at?: string
  updated_at?: string
}

export type ProductImage = {
  id: string
  product_id: string
  url: string
  thumbnail_url?: string | null
  card_url?: string | null
  full_url?: string | null
  is_primary: boolean
}

export type ProductDetail = AdminProduct & {
  category_slug?: string | null
  avg_rating: number
  review_count: number
  images: ProductImage[]
}

export type Category = {
  id: string
  name: string
  slug: string
  parent_id?: string | null
  children?: Category[]
}

export type Brand = {
  id: string
  name: string
}

// Shipping method codes are admin-managed (see DeliveryOption), so this is an
// open string rather than a fixed union. 'standard'/'express' are the seeded
// defaults.
export type ShippingMethod = string

export type DeliveryOption = {
  id: string
  code: string
  label: string
  price_cents: number
  eta_label: string
  sort_order: number
  active: boolean
}

export type CheckoutAddress = {
  recipient_name: string
  line1: string
  line2?: string
  city: string
  region: string
  postal_code: string
  country: string
}

export type CheckoutRequest = {
  email: string
  phone?: string
  shipping_method: ShippingMethod
  address: CheckoutAddress
  /** Set when the user has already seen an address-verification failure and
   *  chose "Use this address anyway". */
  address_override?: boolean
}

export type Order = {
  id: string
  order_number: string
  user_id?: string | null
  guest_session_id?: string | null
  status: string
  email: string
  phone?: string
  subtotal_cents: number
  shipping_cents: number
  total_cents: number
  shipping_method: ShippingMethod
  created_at: string
  updated_at: string
}

export type OrderItem = {
  id: string
  order_id: string
  product_id?: string | null
  product_name: string
  thumbnail_url?: string | null
  unit_price_cents: number
  quantity: number
  created_at: string
}

export type OrderItemPreview = {
  product_id?: string | null
  product_name: string
  thumbnail_url?: string | null
}

export type OrderResponse = {
  order: Order
  items: OrderItem[]
  address: CheckoutAddress
}

export type CheckoutPrefill = {
  email: string
  phone?: string
  shipping_method: ShippingMethod
  address: CheckoutAddress
}

export type OrderSummary = {
  id: string
  order_number: string
  status: string
  total_cents: number
  item_count: number
  item_previews: OrderItemPreview[]
  created_at: string
}

export type StripeConfig = {
  publishable_key: string
}

export type CreateIntentResponse = {
  client_secret: string
  publishable_key: string
  payment_intent_id: string
}

export type SavedAddress = {
  id: string
  label?: string | null
  recipient_name: string
  line1: string
  line2?: string
  city: string
  region: string
  postal_code: string
  country: string
  is_default: boolean
  created_at: string
}

export type SavedAddressInput = {
  label?: string
  recipient_name: string
  line1: string
  line2?: string
  city: string
  region: string
  postal_code: string
  country: string
  is_default?: boolean
}

export type AdminMe = { role: string; email: string }

export type AdminOrderSummary = OrderSummary & {
  email: string
  is_guest: boolean
}

export type AdminOrderList = {
  orders: AdminOrderSummary[]
  total: number
  page: number
  page_size: number
}

export type AdminUser = {
  id: string
  email: string
  role: string
  created_at: string
}

export type AdminUserList = {
  users: AdminUser[]
  total: number
  page: number
  page_size: number
}

export type UserRole = 'admin' | 'support' | 'sales' | 'customer'

export type ModerationReview = {
  id: string
  product_id: string
  product_name: string
  rating: number
  comment?: string | null
  status: 'pending' | 'approved' | 'hidden'
  created_at: string
}

export type BulkUploadResult = {
  created: number
  failed: number
  errors: { index: number; name?: string; error: string }[]
}

export type AuditEntry = {
  id: number
  actor_email?: string | null
  actor_role?: string | null
  action: string
  target: string
  status_code: number
  ip?: string | null
  occurred_at: string
}

export const api = {
  listProducts: (params?: {
    categoryId?: string
    q?: string
    page?: number
    pageSize?: number
    onSale?: boolean
    sort?: string
    brandId?: string
    minPrice?: number
    maxPrice?: number
    minRating?: number
  }) => {
    const qs = new URLSearchParams()
    if (params?.categoryId) qs.set('category_id', params.categoryId)
    if (params?.q) qs.set('q', params.q)
    if (params?.page) qs.set('page', String(params.page))
    if (params?.pageSize) qs.set('page_size', String(params.pageSize))
    if (params?.onSale) qs.set('on_sale', 'true')
    if (params?.sort) qs.set('sort', params.sort)
    if (params?.brandId) qs.set('brand_id', params.brandId)
    if (params?.minPrice != null) qs.set('min_price', String(params.minPrice))
    if (params?.maxPrice != null) qs.set('max_price', String(params.maxPrice))
    if (params?.minRating != null) qs.set('min_rating', String(params.minRating))
    const s = qs.toString()
    return request<ProductsResponse>(`/api/v1/products${s ? `?${s}` : ''}`)
  },
  suggestProducts: (q: string) =>
    request<ProductSuggestion[]>(`/api/v1/products/suggest?q=${encodeURIComponent(q)}`),
  getCart: () => request<Cart>('/api/v1/cart'),
  addItem: (productId: string, quantity: number) =>
    request<Cart>('/api/v1/cart/items', {
      method: 'POST',
      body: { product_id: productId, quantity },
    }),
  updateItem: (productId: string, quantity: number) =>
    request<Cart>(`/api/v1/cart/items/${productId}`, {
      method: 'PUT',
      body: { quantity },
    }),
  removeItem: (productId: string) =>
    request<Cart>(`/api/v1/cart/items/${productId}`, { method: 'DELETE' }),
  clearCart: () =>
    request<{ message: string }>('/api/v1/cart', { method: 'DELETE' }),
  login: (email: string, password: string, totp_code?: string) =>
    request<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { email, password, totp_code },
    }),
  refresh: () =>
    request<LoginResponse>('/api/v1/auth/refresh', { method: 'POST' }),
  logout: () =>
    request<{ message: string }>('/api/v1/auth/logout', { method: 'POST' }),
  register: (email: string, password: string, captchaToken?: string) =>
    request<{ message: string }>('/api/v1/auth/register', {
      method: 'POST',
      body: { email, password, captcha_token: captchaToken ?? '' },
    }),
  forgotPassword: (email: string) =>
    request<{ message: string }>('/api/v1/auth/forgot-password', {
      method: 'POST',
      body: { email },
    }),
  resetPassword: (token: string, new_password: string) =>
    request<{ message: string }>('/api/v1/auth/reset-password', {
      method: 'POST',
      body: { token, new_password },
    }),
  setup2FA: () =>
    request<TwoFactorSetupResponse>('/api/v1/auth/2fa/setup', {
      method: 'POST',
    }),
  enable2FA: (code: string) =>
    request<{ message: string }>('/api/v1/auth/2fa/enable', {
      method: 'POST',
      body: { code },
    }),
  disable2FA: (code: string) =>
    request<{ message: string }>('/api/v1/auth/2fa/disable', {
      method: 'POST',
      body: { code },
    }),
  oauthRedirectUrl: (provider: 'google' | 'facebook') =>
    `/api/v1/auth/oauth/${provider}`,
  // Admin — product CRUD + categories + brands.
  adminListProducts: () => request<ProductsResponse>('/api/v1/products?page_size=100'),
  getProduct: (id: string) => request<ProductDetail>(`/api/v1/products/${id}`),
  adminCreateProduct: (body: Partial<AdminProduct>) =>
    request<AdminProduct>('/api/v1/admin/products', { method: 'POST', body }),
  adminUpdateProduct: (
    id: string,
    body: Partial<AdminProduct> & { clear_sale_price?: boolean },
  ) =>
    request<AdminProduct>(`/api/v1/admin/products/${id}`, { method: 'PUT', body }),
  adminDeleteProduct: (id: string) =>
    request<void>(`/api/v1/admin/products/${id}`, { method: 'DELETE' }),
  adminAddProductImage: (productId: string, url: string, isPrimary = false) =>
    request<{ id: string }>(`/api/v1/admin/products/${productId}/images`, {
      method: 'POST',
      body: { url, is_primary: isPrimary },
    }),
  adminDeleteProductImage: (productId: string, imageId: string) =>
    request<void>(`/api/v1/admin/products/${productId}/images/${imageId}`, {
      method: 'DELETE',
    }),
  adminUploadProductImage: async (
    productId: string,
    file: File,
    isPrimary = false,
  ): Promise<ProductImage> => {
    const form = new FormData()
    form.append('image', file)
    form.append('is_primary', String(isPrimary))
    const res = await fetch(`/api/v1/admin/products/${productId}/images/upload`, {
      method: 'POST',
      credentials: 'include',
      headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
      body: form,
    })
    const text = await res.text()
    const data = text ? JSON.parse(text) : null
    if (!res.ok) {
      throw new ApiError(res.status, data?.error ?? `upload failed (${res.status})`, data?.code)
    }
    return data as ProductImage
  },
  listCategories: () => request<Category[]>('/api/v1/categories'),
  getCategoryBySlug: (slug: string) =>
    request<Category>(`/api/v1/categories/${encodeURIComponent(slug)}`),
  adminCreateCategory: (body: { name: string; slug: string; parent_id?: string }) =>
    request<Category>('/api/v1/admin/categories', { method: 'POST', body }),
  adminUpdateCategory: (id: string, body: { name: string; slug: string; parent_id?: string }) =>
    request<Category>(`/api/v1/admin/categories/${id}`, { method: 'PUT', body }),
  adminDeleteCategory: (id: string) =>
    request<void>(`/api/v1/admin/categories/${id}`, { method: 'DELETE' }),
  listDeliveryOptions: () => request<DeliveryOption[]>('/api/v1/delivery-options'),
  adminListDeliveryOptions: () =>
    request<DeliveryOption[]>('/api/v1/admin/delivery-options'),
  adminCreateDeliveryOption: (body: {
    code: string
    label: string
    price_cents: number
    eta_label: string
    sort_order: number
    active?: boolean
  }) => request<DeliveryOption>('/api/v1/admin/delivery-options', { method: 'POST', body }),
  adminUpdateDeliveryOption: (
    id: string,
    body: { label: string; price_cents: number; eta_label: string; sort_order: number; active?: boolean },
  ) => request<DeliveryOption>(`/api/v1/admin/delivery-options/${id}`, { method: 'PUT', body }),
  adminDeleteDeliveryOption: (id: string) =>
    request<void>(`/api/v1/admin/delivery-options/${id}`, { method: 'DELETE' }),
  listBrands: () => request<Brand[]>('/api/v1/brands'),
  adminCreateBrand: (body: { name: string }) =>
    request<Brand>('/api/v1/admin/brands', { method: 'POST', body }),
  mergeStatus: () => request<MergeStatus>('/api/v1/cart/merge-status'),
  merge: (strategy: 'guest' | 'user') =>
    request<Cart>('/api/v1/cart/merge', {
      method: 'POST',
      body: { strategy },
    }),
  checkout: (req: CheckoutRequest) =>
    request<OrderResponse>('/api/v1/checkout', {
      method: 'POST',
      body: req,
    }),
  getCheckoutPrefill: async (): Promise<CheckoutPrefill | null> => {
    try {
      const data = await request<CheckoutPrefill | null>('/api/v1/checkout/prefill')
      return data ?? null
    } catch (e) {
      // Guests and signed-out users can't prefill — fail silently so the
      // checkout form just renders empty rather than showing an error.
      if (e instanceof ApiError && e.status === 401) return null
      throw e
    }
  },
  listOrders: (params?: { status?: string; from?: string; to?: string }) => {
    const q = new URLSearchParams()
    if (params?.status) q.set('status', params.status)
    if (params?.from) q.set('from', params.from)
    if (params?.to) q.set('to', params.to)
    const qs = q.toString()
    return request<OrderSummary[]>(`/api/v1/orders${qs ? `?${qs}` : ''}`)
  },
  getOrder: (id: string) => request<OrderResponse>(`/api/v1/orders/${id}`),
  cancelOrder: (id: string) =>
    request<OrderResponse>(`/api/v1/orders/${id}/cancel`, { method: 'POST' }),
  getStripeConfig: () => request<StripeConfig>('/api/v1/config/stripe'),
  getDemoConfig: () => request<{ demo: boolean }>('/api/v1/config/demo'),
  createPaymentIntent: (orderId: string) =>
    request<CreateIntentResponse>(`/api/v1/orders/${orderId}/payment-intent`, {
      method: 'POST',
    }),
  recommendations: (limit = 4) =>
    request<{ items: ProductListItem[] }>(`/api/v1/recommendations?limit=${limit}`),
  submitContact: (body: { name: string; email: string; subject: string; message: string }) =>
    request<{ message: string }>('/api/v1/contact', { method: 'POST', body }),
  // --- Saved addresses ---
  listAddresses: () =>
    request<{ addresses: SavedAddress[] }>('/api/v1/addresses'),
  createAddress: (body: SavedAddressInput) =>
    request<SavedAddress>('/api/v1/addresses', { method: 'POST', body }),
  updateAddress: (id: string, body: SavedAddressInput) =>
    request<SavedAddress>(`/api/v1/addresses/${id}`, { method: 'PUT', body }),
  deleteAddress: (id: string) =>
    request<{ message: string }>(`/api/v1/addresses/${id}`, { method: 'DELETE' }),
  setDefaultAddress: (id: string) =>
    request<{ message: string }>(`/api/v1/addresses/${id}/default`, { method: 'POST' }),
  // --- Reviews ---
  listReviews: (productId: string, sort: ReviewSort = 'helpful', page = 1) => {
    const q = new URLSearchParams({ sort, page: String(page) })
    return request<ReviewListResult>(`/api/v1/products/${productId}/reviews?${q}`)
  },
  reviewEligibility: (productId: string) =>
    request<ReviewEligibility>(`/api/v1/products/${productId}/reviews/eligibility`),
  createReview: (productId: string, rating: number, comment?: string) =>
    request<PublicReview>(`/api/v1/products/${productId}/reviews`, {
      method: 'POST',
      body: { rating, comment: comment || undefined },
    }),
  voteHelpful: (reviewId: string, helpful: boolean) =>
    request<{ voted: boolean }>(`/api/v1/reviews/${reviewId}/helpful`, {
      method: helpful ? 'POST' : 'DELETE',
    }),
  // --- Admin ---
  adminMe: () => request<AdminMe>('/api/v1/admin/me'),
  adminListOrders: (params?: { status?: string; page?: number }) => {
    const q = new URLSearchParams()
    if (params?.status) q.set('status', params.status)
    if (params?.page) q.set('page', String(params.page))
    const s = q.toString()
    return request<AdminOrderList>(`/api/v1/admin/orders${s ? `?${s}` : ''}`)
  },
  adminGetOrder: (id: string) => request<OrderResponse>(`/api/v1/admin/orders/${id}`),
  adminUpdateOrderStatus: (id: string, status: string) =>
    request<OrderResponse>(`/api/v1/admin/orders/${id}/status`, {
      method: 'PATCH',
      body: { status },
    }),
  adminRefundOrder: (id: string) =>
    request<OrderResponse>(`/api/v1/admin/orders/${id}/refund`, { method: 'POST' }),
  adminListUsers: (page = 1) =>
    request<AdminUserList>(`/api/v1/admin/users?page=${page}`),
  adminSetUserRole: (id: string, role: UserRole) =>
    request<{ role: string }>(`/api/v1/admin/users/${id}/role`, {
      method: 'PATCH',
      body: { role },
    }),
  adminListReviews: (status?: string) => {
    const q = status ? `?status=${status}` : ''
    return request<{ reviews: ModerationReview[]; total: number }>(`/api/v1/admin/reviews${q}`)
  },
  adminSetReviewStatus: (id: string, status: 'pending' | 'approved' | 'hidden') =>
    request<{ status: string }>(`/api/v1/admin/reviews/${id}`, {
      method: 'PATCH',
      body: { status },
    }),
  adminDeleteReview: (id: string) =>
    request<void>(`/api/v1/admin/reviews/${id}`, { method: 'DELETE' }),
  adminBulkUploadJSON: (products: Partial<AdminProduct>[]) =>
    request<BulkUploadResult>('/api/v1/admin/products/bulk', {
      method: 'POST',
      body: products,
    }),
  adminBulkUploadCSV: async (csv: string): Promise<BulkUploadResult> => {
    const res = await fetch('/api/v1/admin/products/bulk', {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'text/csv',
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
      body: csv,
    })
    const text = await res.text()
    const data = text ? JSON.parse(text) : null
    if (!res.ok) {
      const msg = data?.error ?? `upload failed (${res.status})`
      throw new ApiError(res.status, msg, data?.code)
    }
    return data as BulkUploadResult
  },
  adminListAudit: (page = 1) =>
    request<{ entries: AuditEntry[]; total: number }>(`/api/v1/admin/audit-log?page=${page}`),
}

export function formatPrice(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

// discountPercent returns the rounded percentage off when a valid sale price is
// set, or null when there's no active discount. Used for deal badges.
export function discountPercent(
  price: number,
  salePrice?: number | null,
): number | null {
  if (salePrice == null || salePrice <= 0 || salePrice >= price) return null
  return Math.round(((price - salePrice) / price) * 100)
}
