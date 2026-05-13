# PR 4 — Page audit & edge-case cleanup

**Goal:** Walk every page in both light and dark mode. Catch inconsistencies, fix them as **targeted small diffs**. Resist the urge to redesign anything.

**Risk:** Medium. The audit is mechanical, but the temptation to "improve while you're in there" is high. Reject that temptation.

**Files this PR might touch:** any `*Page.tsx`. Any modal. Any state-dependent UI.

**Files this PR should NOT touch:**
- `tokens.css`, `index.css`
- `components/*.tsx` — those were finished in PR 3
- Any business logic, API call, route, validator
- The editorial product grid in `ProductsPage.tsx`, the cart panel in `CartPanel.tsx`, or the admin pages — **these are explicitly out of scope per the design system**. Leave them as they are.

---

## Prompt to paste into Claude Code

> Read `docs/design-system/README.md` end-to-end before starting.
>
> Audit every page in `web/src/` against the spec. For each page, open it in the browser in **both light and dark mode**, click every interactive element, and check:
>
> 1. **Hardcoded colors.** Search the codebase for hex codes (`#`), `rgb(`, `text-red-`, `text-emerald-`, `text-amber-`, `text-sky-`, `bg-gray-`, `border-gray-`, `bg-black`, `text-black`, `text-white`. Replace each with the closest semantic token (`text-accent` for errors, `text-ink-faint` for muted, `border-rule` for hairlines, etc). The status badges in `OrderStatus.tsx` are the main offender — they use raw Tailwind palette colors. Either map them to semantic tokens or accept the mismatch and document it as a TODO.
>
> 2. **Sentence case.** Buttons and headings should be sentence case, ending in a period for action-shaped copy ("Log in.", "Continue to payment", "Add to cart"). The spec is strict on this. Search for `Title Case Buttons`, all-caps copy, and exclamation marks.
>
> 3. **Focus rings.** Every `:focus-visible` should be the accent ring. The token-level CSS handles this globally — if any component overrides it with `focus:ring-blue` or similar, remove the override.
>
> 4. **Error states.** Error text uses `text-accent` (per the spec). Search for `text-red-`, `text-rose-`, `bg-red-`. The checkout server-error banner in `CheckoutPage.tsx` uses `border-red-200 bg-red-50 text-red-800` — replace with semantic tokens or accept the mismatch.
>
> 5. **Numbers and prices.** Anything numeric should have the `tnum` class or `font-variant-numeric: tabular-nums`. Search for prices, counts, dates, ETAs — make sure they're tabular.
>
> 6. **Inputs.** Confirm every `<input>`, `<select>`, `<textarea>` uses the underline pattern, not a box. The admin pages have inline `border-gray-300` inputs — replace with the project's `Field` component or the equivalent classes.
>
> 7. **Dark mode.** Every page should be legible in dark mode. Look for hardcoded `text-gray-700` or similar that breaks in dark mode. Replace with semantic tokens.
>
> For each finding, make the smallest possible diff. **Do not refactor the file. Do not move components around. Do not rename anything.** One commit per page audited (or per file changed), so reviewers can read the diff cleanly.
>
> If you find a pattern that the spec doesn't cover (e.g., a tooltip, a popover, a toast), stop and **list it in `docs/design-system/TODO.md`** instead of inventing styling for it.
>
> Out of scope (do not touch):
> - `ProductsPage.tsx` editorial grid layout
> - `CartPanel.tsx` slide-out drawer
> - Admin pages
> - Any motion in `motion.tsx`

## Definition of done

- [ ] No raw `#hex`, `rgb()`, or Tailwind palette colors (`text-red-`, `bg-gray-`, etc.) remain in user-facing components.
- [ ] All action buttons are sentence case.
- [ ] No exclamation marks in user-facing copy.
- [ ] All prices, counts, dates have `tnum`.
- [ ] No raw `<input>` with a box style — every form field uses the `Field` component or its underline equivalent.
- [ ] Click through Shop → Cart → Checkout → Pay → Order detail → Order history → Account → Login → Register → Forgot password, in both light and dark mode. No visual breakage.
- [ ] `docs/design-system/TODO.md` lists any pattern that was found but is unspecified.

## Stop conditions

If you find yourself wanting to:

- Add a new pattern to the design system → stop, list in TODO, file as a separate design pass.
- "Modernize" a page beyond the spec → stop, the spec is intentional.
- Touch the cart panel or admin layout → stop, those are out of scope.
- Rewrite a component instead of patching it → stop, small diffs only.

## After it lands

You're done. The codebase now matches the Stora design system. Tag the merge as `design-system-v1`.

Future work:

- Decide whether to fold the editorial product grid, cart panel, and admin chrome into the design system as a v2. If yes, that's a new design pass, not Claude Code work.
- Decide on real product photography to replace the gradient placeholders.
