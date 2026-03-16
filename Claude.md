# i-love-shopping — Claude Code Project Guide

## Project Overview

A full-scale B2C e-commerce platform for technology products (electronics, gadgets, devices).
Think Gigantti or Power.fi in terms of professionalism, UX quality, and frontend polish.
This is a school project but must be built to real-world production standards.

---

## Tech Stack

- **Backend**: Go (Chi router, pgx/v5, golang-migrate)
- **Database**: PostgreSQL (via Docker)
- **Auth**: JWT (access + refresh tokens), bcrypt, UUID
- **Architecture**: Clean layered — handler → service → repository
- **Containerization**: Docker + docker-compose (single command startup)
- **Frontend**: Professional, production-grade UI — Gigantti/Power.fi quality level

---

## Architecture Rules

- **Handler layer**: HTTP only — routing, request parsing, response writing. Zero business logic.
- **Service layer**: Business logic only. No SQL, no HTTP concerns.
- **Repository layer**: Database access only. Returns domain errors, never raw DB errors.
- **Never skip layers.** A handler never calls the repository directly.
- Interfaces for all dependencies — every service and repository must be behind an interface.

---

## Code Style

- Explicit error handling — no silent failures, no ignored errors
- Domain errors defined and mapped at the repository layer
- Follow Go standard project layout conventions
- Raw SQL with pgx — no ORMs
- All new features need tests before they are considered done

---

## Frontend Standards

The frontend must look and feel like a real, professional Finnish tech e-commerce store (Gigantti / Power.fi level). This means:

- Clean, modern product listing pages with filters and sorting
- Polished product detail pages with images, specs, pricing, stock status
- Smooth, intuitive cart and checkout flow
- Responsive design — mobile and desktop
- Professional typography, color system, and component library
- No generic AI-looking UI — it must look like a real commercial product
- Consistent design system across all pages (spacing, colors, fonts, buttons)
- Loading states, error states, and empty states handled everywhere

---

## Authentication Requirements

- Email + password registration and login
- OAuth (Google and/or Facebook)
- CAPTCHA on registration (Google reCAPTCHA)
- JWT: short-lived access tokens (15–60 min), longer refresh tokens (3–7 days)
- **Access tokens stored in memory only** — never localStorage or sessionStorage
- Refresh token rotation: single-use, new token issued on every refresh, old tokens rejected
- Token revocation mechanism for both access and refresh tokens
- Password recovery and reset via email
- Optional user-enabled 2FA (Google Authenticator / Authy)
- Input validation on both client and server side with helpful error messages

---

## Product Catalog Requirements

### Data Model (all fields required)

- `id`, `name`, `description`, `price`, `stock_quantity`
- `category`, `brand`, `images`
- `weight` and `dimensions` in both metric and imperial units

### Features

- Products organized into categories with intuitive browsing
- Faceted search: filter by price range, brand, category, ratings
- Dynamic search suggestions as users type
- Sorting: relevance, price (asc/desc), ratings
- Product images with proper file handling and serving

---

## Database Requirements

- PostgreSQL with full ACID compliance
- Designed for scale — anticipate high traffic during peak periods
- ERD must be documented in README with:
  - Entities, attributes, relationships
  - Primary keys, foreign keys
  - Cardinality and modality
- Efficient queries — use indexes where appropriate
- Caching strategy considered for read-heavy operations

---

## Testing Requirements

### Automated (run before every commit)

**Unit Tests**

- JWT token generation, validation, and expiration
- User input validation (various scenarios including edge cases)
- Product data model validation

**API Integration Tests**

- All endpoints: correct responses and error handling
- Database operations: data persistence and retrieval

**Security Tests**

- Input validation against injection attacks
- Malformed input handling

### Manual Tests (run periodically)

- CAPTCHA verification — proper integration and UX
- OAuth flow — seamless third-party authentication
- 2FA — setup process and login flow with 2FA enabled

### Rules

- TDD where possible — write tests before or alongside implementation
- Use established Go testing libraries (testify, etc.)
- No feature is done without tests

---

## Mandatory Review Checklist

- [ ] README: project overview, ERD, setup instructions, usage guide
- [ ] B2C e-commerce model implemented
- [ ] Email-password + OAuth authentication
- [ ] CAPTCHA on registration
- [ ] JWT concepts understood and correctly implemented
- [ ] Access tokens in memory only
- [ ] Refresh token rotation with single-use validation (old tokens rejected)
- [ ] Token revocation for both token types
- [ ] Password recovery via email
- [ ] Optional 2FA implemented
- [ ] Input validation client + server side
- [ ] ERD provided and documented
- [ ] Product model has all required fields including metric + imperial dimensions
- [ ] Categories with browsing structure
- [ ] Faceted search implemented
- [ ] Sorting options (relevance, price, rating)
- [ ] Product images handled and served
- [ ] Automated tests: unit, API integration, security
- [ ] Architecture is explainable and justified
- [ ] Docker: single startup command runs the entire application

---

## Docker Requirements

- `Dockerfile` (or multiple for frontend/backend separation)
- `docker-compose.yml` that starts everything with one command
- Docker is the **only** host prerequisite — all dependencies run inside containers
- Include a clear startup command in README (e.g., `docker-compose up --build`)

---

## Current Build State

> Update this section as features are completed.

- [ ] Project scaffolding and Docker setup
- [ ] Database schema + migrations
- [ ] User registration + login (email/password)
- [ ] OAuth integration
- [x] JWT auth system (access + refresh + revocation)
- [ ] 2FA
- [ ] Password reset flow
- [ ] Product catalog (model + repository + service + handler)
- [ ] Search + faceted filtering
- [ ] Frontend — product listing page
- [ ] Frontend — product detail page
- [ ] Frontend — cart
- [ ] Frontend — checkout
- [ ] Frontend — auth pages
- [ ] Testing suite
- [ ] README + ERD
