# PR 3 — Component refinements

**Goal:** Make `Button`, `Field`, `Masthead`, and `Nav` match the spec exactly. These are already close — this PR is small focused diffs, not a rewrite.

**Risk:** Medium. Touches components used everywhere. Manually click every page after each commit.

**Files this PR should touch:**
- `web/src/components/Button.tsx`
- `web/src/components/Field.tsx`
- `web/src/components/Masthead.tsx`
- `web/src/components/Nav.tsx`

**Files this PR should NOT touch:** any page (`*Page.tsx`). Any token. Any motion primitive (`lib/motion.tsx`).

---

## Prompt to paste into Claude Code

> Read `docs/design-system/README.md` for context, especially the sections on **Buttons**, **Forms**, **Type**, and the **Components — Buttons / Field / Nav / Masthead** preview cards.
>
> Audit `Button.tsx`, `Field.tsx`, `Masthead.tsx`, `Nav.tsx` against the spec. For each, make only the changes the spec requires. Do not "improve" anything that already matches.
>
> Specifically check:
>
> **`Button.tsx`**
> - `bg-accent` on primary, `bg-accent-soft` on hover. (No `bg-black`.)
> - Ghost variant: 1px `border-rule-strong`, hovers to `border-ink` + `bg-sunken`.
> - Link variant: `text-ink-soft`, `hover:text-ink`, `underline underline-offset-4`. No background.
> - All three variants are `border-radius: 0`. Tailwind default is fine — just make sure no rounded class snuck in.
>
> **`Field.tsx`**
> - Input has **no top/left/right border**, only `border-b border-rule-strong`. `border-radius: 0`.
> - Focus: `border-color: var(--color-ink)` on the bottom edge only.
> - Label is `uc-tight text-[0.7rem] text-ink-faint mb-2`.
> - Error text is `text-accent`, `text-xs`, `mt-1.5`.
> - Hint text is `text-ink-faint`, `text-xs`, `mt-1.5`.
>
> **`Masthead.tsx`**
> - The numeric marker + eyebrow row is `uc-tight text-[0.7rem]` with a `text-rule-strong` `/` separator between number and eyebrow.
> - Title uses `font-display`. After the font swap in PR 2, this should be `font-bold` (700) — the spec lists weight 700 with tracking `-0.02em` and leading `0.95`. If `fontVariationSettings` is still here, remove it.
> - Caption is `text-ink-soft max-w-[55ch] text-[0.95rem] leading-relaxed`.
>
> **`Nav.tsx`**
> - Wordmark already done in PR 2. Verify it survived.
> - Active route gets a 1px accent underline via `after:` (already in `activeStyle`). Confirm the underline color is `bg-accent`, not `bg-ink`.
> - The Cart count `— N` is `text-ink-faint tnum`. After the count changes, it should pulse (scale 1.08 for 240ms) — this is **already implemented** per the codebase audit. Leave it alone.
> - Theme toggle stays as-is.
>
> Don't touch:
> - `motion.tsx`
> - Any `*Page.tsx`
> - `ThemeToggle.tsx`
> - Anything in `admin/`, `cart/`, `checkout/`, `orders/`, `account/`, `auth/`
>
> Don't introduce new components. Don't add new variants. Don't add new props. Don't refactor the file structure.

## Definition of done

- [ ] Primary button is terracotta with white text, square corners, hovers to a slightly darker terracotta.
- [ ] Ghost button has a 1px rule, no fill, hovers to ink border + sunken background.
- [ ] Link button is underlined text, no background.
- [ ] Fields have only a bottom border, no box.
- [ ] Field focus darkens only the bottom border.
- [ ] Masthead heading is Lato bold, leading `0.95`.
- [ ] No `fontVariationSettings` remain in these 4 files.
- [ ] No console errors.
- [ ] All pages still render (click through Shop, Cart, Checkout, Orders, Account, Login).

## Known weirdness (expected)

- Page-level pages (`ProductsPage`, `CheckoutPage`, etc.) may still have inconsistencies — those are addressed in PR 4 as targeted fixes, not a rewrite.

## After it lands

Commit, merge, then move to `pr-04-audit.md`.
