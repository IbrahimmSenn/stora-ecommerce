# Storefront (React 19 + TypeScript + Vite)

The customer-facing SPA and admin dashboard for I Love Shopping. Tailwind v4,
custom OKLCH design tokens, light/dark theme, Stripe Elements checkout. Served
by the Go API in production (`web/dist` is baked into the Docker image); in
development it runs on Vite with an `/api` proxy to the backend.

## Commands

```bash
npm run dev      # Vite dev server on :5173, proxies /api to :8080
npm run build    # type-check (tsc -b, strict) + production build
npm test         # Vitest suite
npm run lint     # ESLint — errors fail CI
```

Run `make up` at the repo root first so the API is available on :8080. There is
no frontend `.env`: the API is same-origin (`/api/v1/...`), and runtime config
(Stripe publishable key, reCAPTCHA site key) is fetched from the backend.

## Layout

```
src/
  lib/         typed API client (ApiError, request timeout, auth header), theme, motion
  components/  shared UI (Nav, Seo, ErrorBoundary, Skeleton, SidePanel, ...)
  products/    PLP + PDP
  cart/        cart page, slide-in panel, recommendations rail
  checkout/    single-page checkout with Stripe PaymentElement
  payment/     payment retry page, Stripe appearance + error mapping
  orders/      history, detail / confirmation
  auth/        login, register, password reset, OAuth callback, 2FA
  account/     profile, addresses, 2FA management
  admin/       role-gated dashboard (products, orders, users, reviews, audit, ...)
  pages/       about, contact
```

State is React Context split by concern (auth, cart, cart panel, toast, theme),
each with a dedicated hook. Per-page `<title>`/meta/OG tags use React 19's
native head hoisting via `components/Seo.tsx` — no helmet dependency.
