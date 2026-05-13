# PR 2 — Swap fonts, wordmark, favicon

**Goal:** Replace Bricolage Grotesque with Lato. Replace the "i-love-shopping" wordmark with `STORA`. Replace the favicon.

**Risk:** Low. Mechanical replacements with one visible side effect (font swap).

**Files this PR should touch:**
- `web/src/main.tsx` (or wherever `@fontsource-variable/bricolage-grotesque` is imported)
- `web/package.json` (remove Bricolage dependency, add Lato)
- `web/src/components/Nav.tsx` (wordmark only)
- `web/index.html` (`<title>`)
- `web/public/favicon.svg` (overwrite from this bundle)

**Files this PR should NOT touch:** any other component. Any token. Any page.

---

## Prompt to paste into Claude Code

> Read `docs/design-system/README.md` for context.
>
> Swap the display font and brand wordmark from "i-love-shopping" / Bricolage Grotesque to "STORA" / Lato.
>
> Steps:
>
> 1. In `web/package.json`, remove the `@fontsource-variable/bricolage-grotesque` dependency. Add `@fontsource/lato` (note: not variable — Lato isn't a variable font). Run `npm install`.
>
> 2. In `web/src/main.tsx` (or wherever fonts are imported), replace the Bricolage import with:
>    ```ts
>    import '@fontsource/lato/400.css'
>    import '@fontsource/lato/700.css'
>    import '@fontsource/lato/900.css'
>    ```
>    Keep the existing Hanken Grotesk imports untouched.
>
> 3. In `web/src/components/Nav.tsx`, change the brand `<Link>` from:
>    ```tsx
>    <Link to="/" className="font-display text-[1.05rem] tracking-tight"
>      style={{ fontVariationSettings: '"wght" 600, "opsz" 24' }}>
>      i-love-shopping
>    </Link>
>    ```
>    to:
>    ```tsx
>    <Link to="/" className="font-display text-[0.95rem] uppercase tracking-[0.32em] font-bold">
>      STORA
>    </Link>
>    ```
>    Note: drop the `fontVariationSettings` style (Lato is not variable). Use `font-bold` (700) instead.
>
> 4. In `web/index.html`, change `<title>i-love-shopping</title>` to `<title>Stora</title>`.
>
> 5. Replace `web/public/favicon.svg` with the file at `docs/design-system/assets/favicon.svg` from this handoff bundle.
>
> 6. Search the codebase for any remaining instance of `i-love-shopping` (string, comment, alt text). If you find any in user-facing copy, replace with `Stora`. Leave `package.json#name` alone — that's the npm package identifier, not the brand.
>
> Then audit `Masthead.tsx` and any other component that uses `fontVariationSettings`. Lato doesn't support variation settings, so those lines either need to be removed or replaced with `fontWeight` declarations:
> - `'"wght" 540, "opsz" 32'` → `fontWeight: 700`
> - `'"wght" 500'` → `fontWeight: 700`
> - `'"wght" 600, "opsz" 24'` → `fontWeight: 700`
>
> Match the spec in `docs/design-system/README.md` § Type: display sets at weight 700.
>
> Do not touch any token CSS. Do not touch any page-level component.

## Definition of done

- [ ] `package.json` no longer mentions Bricolage. Lato is installed.
- [ ] No `fontVariationSettings` remain in `Masthead.tsx`, `Nav.tsx`, or any other component (search the codebase).
- [ ] App renders. Headings are visibly **Lato**, not Bricolage (rounded `g`, more humanist).
- [ ] Nav reads `STORA` in uppercase tracked bold, not "i-love-shopping".
- [ ] Browser tab shows the inked-S favicon, not the gradient lightning glyph.
- [ ] `<title>` is `Stora`.
- [ ] Light and dark mode both still work. No console errors.

## Known weirdness (expected)

- The Masthead title may look slightly heavier than the spec implies. That's because Lato 700 ≈ Bricolage 540 in optical weight. If it looks too heavy at large sizes, the audit PR (PR 4) is the right place to dial the weight down to 600.
- Pages still use the old component spacing — Masthead micro-adjustments come in PR 3.

## After it lands

Commit, merge, then move to `pr-03-components.md`.
