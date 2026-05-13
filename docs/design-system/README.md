# Stora Design System

A design system for **Stora**, the e-commerce project for Kood Sisu coding school.
Stora is a modest, editorial storefront — a small, opinionated shop with a single product surface, cart, checkout, orders, account, and 2FA. The design language is near-monochrome with a terracotta accent and an amber highlight, set in Lato display and Hanken Grotesk body. Sentence-case, calm motion, tabular numerals — the look of a print catalogue made interactive.

> The github repo is **i-love-shopping** (codebase working title); **Stora** is the production brand name used everywhere on-screen.

---

## Sources

- **Codebase (read-only):** `web/` — Vite + React 19 + Tailwind v4 + react-router-dom. Authoritative source for design tokens, fonts, components, and screen patterns.
- **Reference repo:** [`IbrahimmSenn/iloveshopping`](https://github.com/IbrahimmSenn/iloveshopping) (GitHub) — brand exploration.
- **Authoritative tokens:** mirrored at `reference/tokens.css` (from `web/src/styles/tokens.css`, generated per a project doc the repo refers to as `.impeccable.md`).

The reader is **not** assumed to have access to the codebase. Everything required to design for Stora lives in this folder.

---

## Index

| Path | Purpose |
|---|---|
| `README.md` | This file. High-level context + visual + content + iconography rules. |
| `colors_and_type.css` | CSS variables — every color, font, weight, size, radius, motion token. Drop-in. |
| `SKILL.md` | Agent Skill manifest. Read this in any Claude Code / agent flow. |
| `assets/favicon.svg` | Inked S monogram. Favicon contexts only — 16–64px. |
| `assets/icons.svg` | SVG sprite — social icons (X, GitHub, Discord, Bluesky) + utility icons. |
| `fonts/` | Webfont files — Bricolage Grotesque + Hanken Grotesk (variable). |
| `reference/tokens.css` | Verbatim copy of the source `tokens.css` token file for grounding. |
| `preview/` | Design system cards rendered for the Design System tab. |
| `ui_kits/storefront/` | High-fidelity recreation of the Stora storefront. Open `index.html`. |

---

## Product context

Stora is one product: a small storefront. The surfaces are:

- **Shop** (`/`) — product grid, primary image, name, brand, price, stock, "Add to cart".
- **Cart** (`/cart`) — line items with quantity steppers, total, "Checkout".
- **Checkout** (`/checkout`) — three numbered sections (Contact, Shipping address, Shipping method) + sticky summary; ends in a Stripe handoff.
- **Payment** (`/orders/:id/pay`) — Stripe Elements page.
- **Orders** (`/orders`, `/orders/:id`) — list with status filters; detail with line items + status timeline.
- **Account** (`/account`) — security (2FA setup/disable), order history, developer (token tester).
- **Auth** — Log in, Register, Forgot/Reset password, OAuth (Google, Facebook).

There is one consumer audience; admin surfaces are out of scope for the design system.

---

## CONTENT FUNDAMENTALS

**Voice.** Sober, declarative, slightly formal. The product talks like a careful shopkeeper, not a startup. No exclamation marks. No emoji (zero — see Iconography). Sentences end in periods even when short, including buttons and links when they would otherwise read as fragments — e.g. *"Log in."*, *"Set up two-factor."*, *"Open order history."* This is intentional and on-brand; it gives every action the cadence of a complete thought.

**Casing.** Sentence case everywhere — headings, labels, buttons. Uppercase exists only in the `uc-tight` editorial marker style (eyebrows, section numbers, summary labels) — 0.7rem, 0.08em letter-spacing, tabular figures. **Never** title-case headings, **never** all-caps body, **never** `Title Case` buttons.

**Pronouns.** Second person — *"your cart", "your orders", "we'll create your order"*. First person plural ("we") only for the system speaking on its own behalf, sparingly.

**Concision.** A product card is `name → brand → price → stock`, nothing more. The checkout subtitle is *"Review and confirm"* — three words. Captions max ~55ch and only when they earn their place.

**Numbers.** Always tabular (`.tnum` / `font-variant-numeric: tabular-nums`). Prices use the locale formatter (`formatPrice(cents)`). Section markers are zero-padded — `01`, `02`, `03` — never `1.` or `#1`.

**Specific examples from the product:**

- Masthead: eyebrow `Account` · number `01` · title `Log in.` · caption `Password updated. Log in with your new password.`
- Empty cart: `Your cart is empty` / `Nothing here yet.` / `Browse products`
- Checkout sections: `01 Contact` / `02 Shipping address` / `03 Shipping method`
- Submit copy: `Continue to payment` (transitions to `Placing order…` — em-dash + lowercase verb + ellipsis is the pattern for in-flight states)
- 2FA hint: *"6-digit code from your authenticator, or one of your recovery codes."*
- Error tone: short, lowercase, no apology — `invalid email`, `required`, `7–20 characters`, `two-letter code`. Field-level errors are essentially log messages.
- Auth divider: `Or via` — no punctuation, set in `uc-tight`.

**Don'ts.**

- No "Welcome to Stora!" or any greeting copy.
- No exclamation marks.
- No emoji or decorative unicode glyphs.
- No marketing hyperbole (`Amazing`, `Powerful`, `Loved by thousands`).
- No "Click here." Use the verb that names the destination: *"Browse products"*, *"Open token tester."*
- No sentence-starting `So,` or `Just`.

---

## VISUAL FOUNDATIONS

### Colors

OKLCH near-monochrome. **Light mode** is bone tinted toward oxblood (hue 25). **Dark mode** is cool deep-ink (hue 250) with warm ink. **No pure black, no pure white anywhere** — the lightest surface is `oklch(0.99 0.003 25)` and the darkest ink is `oklch(0.18 0.01 25)`.

Three surfaces (`surface-0/1/2`), three inks (`ink`, `ink-soft`, `ink-faint`), two rules (`rule`, `rule-strong`), one accent (`accent`, with `accent-soft` hover and `on-accent` text on top). Feedback colors (`positive`, `warning`, `negative`) are desaturated — `negative` is *the same hue* as `accent` so error states feel native, not alarming.

**Accent is rare.** Used for: primary button background, the focus ring (always visible), the active nav underline, validation errors, and `::selection`. Never for body text emphasis (use `text-ink` weight 540 instead) and never for decorative fills.

### Type

Two variable webfonts:

- **Bricolage Grotesque Variable** — display. Used for: page titles (`clamp(2.25rem, 6vw, 4rem)`, weight 540, opsz 32, tracking -0.02em, leading 0.95), section sub-headings, the wordmark in the nav (weight 600, opsz 24).
- **Hanken Grotesk Variable** — body. Used for: paragraphs, labels, inputs, buttons. Feature settings `"ss01", "cv11"` on `<html>`.

Sizes are intentional and small in number — `text-[0.7rem]` (eyebrows), `text-xs` (errors/hints), `text-sm` (most UI), `text-[0.95rem]` (lead paragraphs), `text-lg / text-xl` (subheads), and the clamped display title.

### Spacing

4pt scale, named `--space-1` through `--space-24` (skipping `5/7/9/10/11/…` — only the values actually used). Generous outer page padding: routes are `py-14 lg:py-20`, container is `px-6 lg:px-10`, masthead has `mb-12 lg:mb-20`. Pages breathe.

### Layout

Editorial, **asymmetric**, **left-aligned**. Container max-widths are content-appropriate: `max-w-md` for auth, `max-w-3xl` for cart/account, `max-w-5xl` default, `max-w-6xl` for the checkout's two-column split. Sections in long forms get a `01/02/03` editorial number + eyebrow + title; in account/admin they become a `[12rem_1fr]` two-column grid with the number+eyebrow on the left.

### Backgrounds

**Flat surfaces.** No gradients, no patterns, no full-bleed photography, no textures. The page background is `--color-surface-0`. Imagery only appears inside product cards, where it's expected to be neutral product photography on a light backdrop.

The favicon glyph is the **only** place a vivid gradient appears, and it is reserved for branding contexts (favicon, tab marker). Do not extend its purple/violet/blue palette into the UI.

### Animation

One easing curve — `--ease-out-quart` `cubic-bezier(0.165, 0.84, 0.44, 1)`. Three durations: `fast 180ms`, `med 360ms`, `slow 560ms`. The only motion primitive is **`<Reveal>`**: first-paint, per-child fade + 8px translateY rise, 80ms stagger. Used on every `<Page>` root. No bounces, no spring physics, no parallax, no Lottie. `prefers-reduced-motion` collapses every duration to `0ms` globally.

### Hover & press states

- **Text links / nav:** `text-ink-soft → text-ink` via `transition-colors`. Active route gets an accent underline (`after:` 1px line bottom-0.5).
- **Primary button:** `bg-accent → bg-accent-soft` (slightly lighter, same family). No scale, no shadow.
- **Ghost button:** `border-rule-strong → border-ink`, plus a `bg-sunken` wash on hover.
- **Cards/rows:** `hover:bg-gray-50` equivalent on rows (orders list); product cards do not lift or shadow.
- **Press:** native browser active state. No deliberate scale-down. **No** ripple, no glow.

### Borders

1px hairline `--color-rule` for separators; `--color-rule-strong` for explicit edges (input underlines, ghost button frames). Borders are square — `--radius-sm: 2px` is the largest radius used on UI. **Buttons, badges and cards are not rounded.** The only radii in use are `2/4/8 px` for tiny chrome (search affordances, etc).

### Shadows

**No shadows.** None. Depth comes from surface-step + 1px rules, never from drop shadow. Cards and modals (e.g. the cart-merge prompt) sit flat on a sibling surface.

### Capsules & badges

Status badges are **flat rectangles** with a 1px border and a tinted background — see `OrderStatus.tsx`. Same convention for any pill: outline + tinted fill, never solid filled + rounded. The accent capsule (focus ring) is the exception and is the only "glow" in the system — 2px outline, 2px offset.

### Transparency & blur

None. Surfaces are opaque. No backdrop-blur, no glassmorphism, no overlay scrims beyond the modal scrim itself (a flat surface, not a blur).

### Corner radii

- `--radius-sm: 2px` — inputs, tiny chrome.
- `--radius-md: 4px` — date picker affordances.
- `--radius-lg: 8px` — rarely used; admin panels at most.
- Default UI radius is **`0`**. Buttons, cards, the cart and orders containers are square.

### Cards

A "card" in Stora is a `border border-rule p-4 flex flex-col` rectangle — flat, square, no shadow. Product cards stack `image → name → brand → price → stock → button`. Order rows are not cards at all — they are 4-column grid rows separated by `divide-y`.

### Forms

Inputs are **underlined**, not boxed (`Field.tsx`): a 1px bottom border `--rule-strong` that darkens to `ink` on focus. `border-radius: 0`. The label is set in `uc-tight` and sits above the input. Hints in `text-ink-faint`, errors in `text-accent` — both `text-xs`, both `mt-1.5`. There is no green checkmark for valid state.

### Focus

Always `outline: 2px solid var(--color-accent); outline-offset: 2px;` via `:focus-visible`. Never removed. Never replaced by a shadow.

### Imagery vibe

Neutral, calm, **no filters**. Product photography on a near-white backdrop (`bg-gray-100` placeholder in the codebase). No warm/cool grade, no film grain, no rounded corners. Aspect ratio is square (`aspect-square object-cover`).

### Layout rules (fixed elements)

- **Nav** is a single hairline-bottom row (not fixed; scrolls with the page). `border-b border-rule`, `max-w-6xl`, `py-5`.
- **Sticky sidebar** in checkout uses `lg:sticky lg:top-8` — no shadow, no border, just sits.
- Modals (cart merge) are centered overlays. No bottom sheets. No drawers.

---

## ICONOGRAPHY

**No emoji. None.** Across the entire codebase there is not a single emoji character in a UI string. Do not introduce any.

**No unicode glyph icons** (no ✓, ★, ↗, etc) in UI copy. Math operators are used for the cart steppers — the **minus sign `−` (U+2212)** and a literal **plus `+`** — set in the body font; treat these as text, not iconography.

**SVG sprite.** The product ships **one** SVG sprite at `web/public/icons.svg`, mirrored here at `assets/icons.svg`. It contains:

- `#bluesky-icon` — Bluesky logo, ink-black fill (`#08060d`).
- `#discord-icon` — Discord wordmark glyph, ink-black fill.
- `#github-icon` — GitHub Octocat, ink-black fill.
- `#x-icon` — X (Twitter) wordmark, ink-black fill.
- `#documentation-icon` — outlined document, 1.35px stroke, purple stroke (`#aa3bff`) — used in the brand exploration repo, not in the main app.
- `#social-icon` — outlined user + star, same purple stroke.

Use the social icons as **inline `<svg><use href="…#bluesky-icon" /></svg>`** at a fixed 16–20px square. Fill icons inherit `#08060d` (effectively `text-ink`); set `fill="currentColor"` after copying if you need theme-aware behavior. Stroked icons keep their purple stroke and should be used only on brand/exploration surfaces.

**There is no general-purpose icon set in the product itself.** The main app uses no icons at all in its UI — actions are named in words ("Add to cart", "Remove", "Continue to payment", "Log in"). Status is named in words ("Pending payment", "Shipped"). Quantity steppers use `−` / `+` as text.

**If you need a UI icon (e.g. for a non-existent feature in a mock):**

1. Prefer extending the sprite at `assets/icons.svg` with a new flat `<symbol>` using the same `#08060d` fill, no stroke, viewBox `0 0 16 16` or `0 0 20 20`.
2. If you need a broader set, use **Lucide** from CDN at `stroke-width: 1.35`, `stroke="currentColor"`, no fill. This matches the stroked icons in the existing sprite and is **flagged as a substitution** — the product itself does not depend on Lucide.
3. Never draw a freehand SVG and never introduce emoji.

**Brand mark.** No symbol — Stora is **wordmark-only**. `assets/favicon.svg` contains an **inked S monogram** for favicon and app-icon contexts (16–64px); above that size, always switch to the wordmark.

**Wordmark.** `S T O R A` set in **Lato Bold (700)**, **uppercase**, letter-spacing `0.32em–0.34em`. Tagline beneath: **“Considered goods, delivered.”** in body type, color `--ink-soft`, no caps treatment. There is no logotype lockup beyond this pair.

---

## Font substitutions

The codebase loads variable fonts from npm (`@fontsource-variable/bricolage-grotesque`, `@fontsource-variable/hanken-grotesk`). Both are also available on **Google Fonts** under the same names — `colors_and_type.css` includes them from Google Fonts so this folder is self-contained without bundling font binaries. If you need pixel-exact metrics, copy `.ttf` files from the npm packages into `fonts/`. **Flag this to the reviewer** if the rendering differs.

---

## Notes for designers

- Lean on the existing surfaces before inventing new patterns. The product is small, the patterns are repeated; consistency is the brand.
- When a layout feels empty, **let it be empty**. The system rewards generous whitespace and punishes filler.
- Numbers (`01`, `02`) before section titles are an editorial signature. Use them whenever a page has more than one section, even if there are only two.
- Sentence case + periods on buttons is the strongest single tell of the brand. Preserve it.
