# PR 1 — Update design tokens

**Goal:** Land ~80% of the visual change by editing only token variables. No component logic touched.

**Risk:** Low. Tailwind v4 `@theme` and CSS custom properties cascade through every component automatically.

**Files this PR should touch:**
- `web/src/styles/tokens.css` (entirely rewritten)
- `web/src/index.css` (only if `@theme` mapping needs new tokens — check first)

**Files this PR should NOT touch:** any `.tsx` file. Any component. Any page.

---

## Prompt to paste into Claude Code

> Read `docs/design-system/README.md` for context.
>
> Update `web/src/styles/tokens.css` to match the **Stora** palette. Replace every value in the `:root` and `.dark` blocks with the OKLCH values below. Keep the file's overall structure, comments, and the `@media (prefers-reduced-motion: reduce)` block at the bottom — only the variable values change.
>
> Also: add a new token `--color-highlight` (light: `oklch(0.88 0.13 85)`, dark: `oklch(0.78 0.13 85)`) and `--color-highlight-ink` (light: `oklch(0.35 0.08 75)`, dark: `oklch(0.20 0.04 75)`) for sale/best-seller badges. Update `--color-on-accent` in light mode to `oklch(0.99 0.003 60)` to match the new surface temperature.
>
> Replace the font stacks at the bottom of `:root`:
> - `--font-display`: was `"Bricolage Grotesque Variable", "Bricolage Grotesque", ui-sans-serif, system-ui, sans-serif`. New: `"Lato", ui-sans-serif, system-ui, sans-serif`.
> - `--font-body`: unchanged (`"Hanken Grotesk Variable", "Hanken Grotesk", ui-sans-serif, system-ui, sans-serif`).
>
> Then open `web/src/index.css` and verify the `@theme` block maps the new tokens. Add `--color-highlight` and `--color-highlight-ink` mappings if the existing block lists named colors. Do not touch anything else in `index.css`.
>
> Do not edit any `.tsx` file in this PR.

## Exact token values (drop in)

### `:root` (light mode)
```css
--color-surface-0: oklch(0.985 0.003 60);
--color-surface-1: oklch(1.00 0 0);
--color-surface-2: oklch(0.95 0.004 60);

--color-ink: oklch(0.18 0.01 25);
--color-ink-soft: oklch(0.42 0.01 25);
--color-ink-faint: oklch(0.58 0.006 25);

--color-rule: oklch(0.88 0.006 25);
--color-rule-strong: oklch(0.78 0.008 25);

--color-accent: oklch(0.62 0.17 38);
--color-accent-soft: oklch(0.55 0.16 38);
--color-on-accent: oklch(0.99 0.003 60);

--color-highlight: oklch(0.88 0.13 85);
--color-highlight-ink: oklch(0.35 0.08 75);

--color-positive: oklch(0.52 0.10 150);
--color-warning: oklch(0.62 0.13 65);
--color-negative: oklch(0.45 0.15 25);
```

### `.dark`
```css
--color-surface-0: oklch(0.14 0.008 250);
--color-surface-1: oklch(0.18 0.008 250);
--color-surface-2: oklch(0.21 0.008 250);

--color-ink: oklch(0.94 0.005 60);
--color-ink-soft: oklch(0.72 0.008 60);
--color-ink-faint: oklch(0.55 0.008 250);

--color-rule: oklch(0.27 0.010 250);
--color-rule-strong: oklch(0.38 0.012 250);

--color-accent: oklch(0.70 0.16 38);
--color-accent-soft: oklch(0.62 0.16 38);
--color-on-accent: oklch(0.14 0.008 250);

--color-highlight: oklch(0.78 0.13 85);
--color-highlight-ink: oklch(0.20 0.04 75);

--color-positive: oklch(0.70 0.12 150);
--color-warning: oklch(0.74 0.13 65);
--color-negative: oklch(0.66 0.16 25);
```

## Definition of done

- [ ] `tokens.css` contains the values above, verbatim.
- [ ] `--font-display` references Lato (not Bricolage).
- [ ] App builds with no TypeScript / lint errors.
- [ ] App still renders. Run `npm run dev` and open the homepage.
- [ ] Buttons are visibly **terracotta**, not oxblood.
- [ ] Page background is visibly **cooler / brighter** (less cream).
- [ ] Dark mode toggle still works and dark mode terracotta is visibly **lighter** than light-mode terracotta.
- [ ] No console errors.

## Known weirdness (expected, not a bug)

- Fonts will still look like Bricolage on screen — that's fixed in **PR 2**. Don't try to fix it here.
- The wordmark "i-love-shopping" is still in the nav — fixed in **PR 2**.
- Sale badges using `--color-highlight` won't appear anywhere yet — that's fine, the token just needs to exist so future code can use it.

## After it lands

Commit, merge, then move to `pr-02-fonts-wordmark.md`.
