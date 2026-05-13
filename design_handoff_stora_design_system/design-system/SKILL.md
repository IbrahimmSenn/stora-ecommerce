---
name: stora-design
description: Use this skill to generate well-branded interfaces and assets for Stora (working title "i-love-shopping"), a small editorial e-commerce storefront for Kood Sisu coding school. Contains essential design guidelines, OKLCH near-monochrome colors with a single oxblood accent, Bricolage Grotesque + Hanken Grotesk type, sentence-case voice rules, the SVG icon sprite, and UI kit components for prototyping.
user-invocable: true
---

Read the README.md file within this skill, and explore the other available files.

If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. Drop `colors_and_type.css` into the page `<head>` to inherit every token in one line.

If working on production code, copy assets and read the rules here to become an expert in designing with this brand. The product's authoritative source-of-truth tokens are mirrored at `reference/tokens.css`.

If the user invokes this skill without any other guidance, ask them what they want to build or design, ask some questions, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

Key rules at a glance — see README.md for full detail:

- **Voice:** sentence case, periods on action copy (`Log in.`), no emoji, no exclamation marks, lowercase error messages.
- **Color:** near-monochrome OKLCH, hue 25 (light) / hue 250 (dark). One oxblood accent used rarely.
- **Type:** Bricolage Grotesque display (weight 540, opsz 32, tracking -0.02em), Hanken Grotesk body, both variable.
- **Radius:** 0 by default. Buttons, cards, badges are square.
- **Shadows:** none. Depth is surface-step + 1px rule.
- **Motion:** one easing curve, three durations, `<Reveal>` first-paint stagger only.
- **Iconography:** the product itself uses **no UI icons** — actions are named in words. Only an SVG sprite of social marks exists. Do not introduce emoji or unicode glyphs.
- **Editorial numbering:** sections start `01`, `02`, `03` zero-padded, set in `uc-tight`.
