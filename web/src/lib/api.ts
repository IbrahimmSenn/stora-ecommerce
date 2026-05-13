// Minimal fetch wrapper. Sends cookies (for guest_session_id) and attaches
// the in-memory access token when one is set by AuthContext.

let accessToken: string | null = null

export function setAccessToken(t: string | null) {
  accessToken = t
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
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
    const msg =
      data &&
      typeof data === 'object' &&
      'error' in data &&
      typeof (data as { error: unknown }).error === 'string'
        ? (data as { error: string }).error
        : `request failed (${res.status})`
    throw new ApiError(res.status, msg)
  }
  return data as T
}

// --- Typed endpoints ---

export type ProductListItem = {
  id: string
  name: string
  price: number
  stock_quantity: number
  category_name?: string | null
  brand_name?: string | null
  primary_image?: string | null
}

export type ProductsResponse = {
  products: ProductListItem[]
  total: number
  page: number
  page_size: number
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
  is_primary: boolean
}

export type ProductDetail = AdminProduct & {
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

export type ShippingMethod = 'standard' | 'express'

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
  unit_price_cents: number
  quantity: number
  created_at: string
}

export type OrderResponse = {
  order: Order
  items: OrderItem[]
  address: CheckoutAddress
}

export type OrderSummary = {
  id: string
  order_number: string
  status: string
  total_cents: number
  item_count: number
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

export const api = {
  listProducts: () => request<ProductsResponse>('/api/v1/products'),
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
  adminUpdateProduct: (id: string, body: Partial<AdminProduct>) =>
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
  listCategories: () => request<Category[]>('/api/v1/categories'),
  adminCreateCategory: (body: { name: string; slug: string; parent_id?: string }) =>
    request<Category>('/api/v1/admin/categories', { method: 'POST', body }),
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
  createPaymentIntent: (orderId: string) =>
    request<CreateIntentResponse>(`/api/v1/orders/${orderId}/payment-intent`, {
      method: 'POST',
    }),
}

export function formatPrice(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}
