# Current Sprint — UI/UX Fixes & Cart Implementation

Append this section to the existing CLAUDE.md. Read `.impeccable.md` and the full CLAUDE.md above before starting any work.

---

## Priority 0: Cart Bug Fix

**Adding items to cart returns 500 Internal Server Error.** This blocks all Project 2 work.

Investigation checklist:
- Check the cart POST endpoint handler — is it parsing the request body correctly?
- Verify `carts` and `cart_items` tables exist and migrations have run.
- Check if the handler expects an authenticated user but the middleware isn't attaching user context (guest cart vs logged-in cart logic may be missing).
- Check repository layer for SQL errors (wrong column names, missing foreign keys, type mismatches).
- Test the endpoint directly with curl, bypassing the frontend, to isolate whether the bug is backend or frontend.
- Do not move to any UI work until add-to-cart works end-to-end.

---

## Navigation Bar Restructuring

### Current state
Text links in a flat row: `STORA  Shop  Cart  Orders  Register  Log In  DARK`. No iconography, no visual hierarchy.

### Changes

**Replace text labels with icons where appropriate:**
- Cart → shopping bag/basket icon with a badge showing item count. No text label.
- Profile/auth → person silhouette icon. Clicking opens a dropdown (Register/Log In when logged out; Account/Log Out when logged in). No text label.
- Keep the "STORA" wordmark on the left as the home link.

**Add a search input** to the nav bar. It should filter products by name and description using the existing faceted search backend from Project 1. Style it to fit the editorial aesthetic from `.impeccable.md` — not a chunky Amazon-style bar. Think minimal: a subtle input that expands on focus.

**Move the theme toggle out of the nav bar.** It moves into the side panel (see below).

**Add a hamburger menu icon (three lines)** on the right side of the nav. This opens the side panel.

All icon choices must be simple, line-weight-consistent SVGs or a single icon set. No mixing icon libraries. Keep it minimal per the "quiet surfaces, loud type" principle.

---

## Side Panel (Hamburger Menu)

Slides in from the right on hamburger click. Semi-transparent backdrop overlay.

### Contents

**Product categories:**
- Derived from the existing product data. Examine what dummy products exist and group them logically.
- Each category links to the shop page filtered to that category.
- Each product in the database needs a `category` field if it doesn't already have one — check the schema first and ask before adding migrations.

**Navigation:**
- Shop (all products)
- Orders

**Theme toggle:**
- Sun icon (current: light mode, click to switch to dark) / Moon icon (current: dark mode, click to switch to light).
- This is the only place the toggle lives. Remove it from everywhere else.

**Auth links** at the bottom (same as profile dropdown content).

### Behavior
- Smooth slide-in transition (respect `prefers-reduced-motion`).
- Close on backdrop click or × button.
- Scrollable if content overflows.

---

## Product Grid Fix

### Problem
Cards in the grid have inconsistent vertical alignment — middle items sit higher than outer ones.

### Fix
- Uniform card heights across the entire grid row. Use CSS Grid with `align-items: stretch` or Flexbox with equal-height logic.
- Product images: consistent aspect ratio container with `object-fit: cover`. No images stretching or squashing.
- If product titles vary in length, ensure the price/action area stays bottom-aligned within each card (flex column with `margin-top: auto` on the bottom section).
- Follow the layout principles from `.impeccable.md` — this does NOT mean identical card grids. The design brief says "no identical card grids." Find a layout that is uniform in alignment but has editorial character (varied spacing, asymmetry where appropriate). Ask if unsure how to reconcile "fix alignment" with "no identical card grids."

### Quick-add button
- Small "+" icon on each product card (bottom-right of image area or card footer).
- Clicking adds 1 unit to cart without navigating away.
- Show a brief, non-blocking toast/notification: "[Product name] added to cart." Auto-dismiss after 3 seconds, slide-in animation (respect `prefers-reduced-motion`).
- If user is not authenticated and guest cart is not yet implemented, show a message prompting login.

---

## Product Detail Page

New route: `/product/:id`

Clicking a product card (anywhere except the "+" button) navigates here.

### Layout — three-column (adapt to editorial style, not a literal Amazon clone)

**Left: Product image(s)**
- Large main image.
- Thumbnail row below if multiple images exist. Click to swap.

**Center: Product information**
- Product name (large, display typeface from `.impeccable.md`).
- Price — use tabular figures as specified in the design brief.
- If discount: original price with strikethrough, discount percentage, current price.
- Full description.
- Specifications as key-value pairs (brand, weight, material, etc.) — only if data exists.

**Right: Purchase box (sticky on scroll)**
- Price.
- Quantity selector (+/- stepper or dropdown).
- "Add to Cart" button (primary accent color from the design system).
- Stock status.
- This column uses `position: sticky; top: ...` to stay visible during scroll.

**Navigation:** Breadcrumb or back link to return to shop.

Style this page according to `.impeccable.md`. The three-column layout is functional guidance, not a mandate to copy Amazon's visual design. Apply editorial composition: asymmetric spacing, typographic hierarchy, generous whitespace.

---

## Implementation Order

1. Fix the 500 error on add-to-cart.
2. Product grid alignment fix.
3. Quick-add "+" button on cards + toast notification.
4. Nav bar restructuring (icons, search, hamburger).
5. Side panel (categories, theme toggle, nav links).
6. Product detail page.
7. Category filtering.
8. Responsive testing and polish.

---

## Reminders

- All existing CLAUDE.md rules still apply (architecture layers, no ORMs, no AI attribution, ask before adding dependencies or migrations).
- All `.impeccable.md` design rules still apply (OKLCH colors, rejected font list, no card-on-card, motion discipline).
- Do not introduce a CSS component library (Bootstrap, Material, Chakra, etc.).
- Do not hardcode hex/rgb colors — use the OKLCH-based design tokens from the design system.
- Icons: use a single consistent icon set. If one is already in the project, use that. If not, ask before adding one.
