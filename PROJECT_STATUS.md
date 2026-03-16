# i-love-shopping — Project Status

## Phase 1: Foundation

- [x] Project scaffolding (Go modules, directory structure, Makefile)
- [x] Database schema + migrations (users, products, categories, brands, images, reviews, refresh tokens, OAuth accounts, password reset tokens, 2FA)
- [ ] Docker setup (Dockerfile, docker-compose.yml, single command startup)

## Phase 2: Authentication

- [x] User registration (email + password) — `POST /api/v1/auth/register`
- [x] User login with JWT token generation — `POST /api/v1/auth/login`
- [x] Refresh token stored in database on login
- [x] Refresh token rotation with single-use validation and replay detection — `POST /api/v1/auth/refresh`
- [x] Logout / token revocation (revokes all user sessions) — `POST /api/v1/auth/logout`
- [x] Auth middleware (validates Bearer token, injects user claims into context)
- [ ] Password reset flow (request reset via email, reset with token)
- [ ] OAuth integration (Google and/or Facebook)
- [ ] CAPTCHA on registration (Google reCAPTCHA)
- [ ] Optional 2FA (Google Authenticator / Authy)

## Phase 3: Product Catalog

- [ ] Product model + repository + service + handler
- [ ] Category browsing structure
- [ ] Brand listing
- [ ] Product images — upload, storage, and serving
- [ ] Product detail endpoint
- [ ] Product listing with pagination

## Phase 4: Search & Filtering

- [ ] Faceted search (filter by price range, brand, category, ratings)
- [ ] Sorting (relevance, price asc/desc, ratings)
- [ ] Dynamic search suggestions

## Phase 5: Frontend

- [ ] Design system (typography, colors, spacing, component library)
- [ ] Auth pages (register, login, password reset, 2FA setup)
- [ ] Product listing page (grid, filters, sorting)
- [ ] Product detail page (images, specs, pricing, stock status)
- [ ] Cart
- [ ] Checkout flow
- [ ] Responsive design (mobile + desktop)
- [ ] Loading states, error states, empty states

## Phase 6: Testing

- [x] Unit tests — JWT generation, validation, expiration, malformed input, signing method attacks
- [x] Unit tests — auth service (login, refresh rotation, replay detection, revocation, logout)
- [x] Unit tests — auth middleware (missing header, invalid format, invalid token, valid token, case insensitivity)
- [ ] Unit tests — user input validation
- [ ] Unit tests — product data model validation
- [ ] API integration tests — all endpoints
- [ ] Security tests — injection attacks, malformed input

## Phase 7: Documentation & Deployment

- [ ] README (project overview, setup instructions, usage guide)
- [ ] ERD diagram (entities, relationships, keys, cardinality)
- [ ] Architecture documentation
