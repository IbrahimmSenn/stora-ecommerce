# Handoff: Stora Design System

## Overview

This package contains the **Stora** brand and design system, ready to be applied to the production codebase at `web/`. The system is near-monochrome editorial commerce with a terracotta accent — Lato display, Hanken Grotesk body, square corners, no shadows.

## How to use this package

**Do not** ask Claude Code to implement everything in one prompt. The codebase is real, has working features, and the design change touches many files. A one-shot rewrite is unreviewable and will break things silently.

Instead, run the four PRs below **in order**, each as a separate Claude Code session and a separate pull request. Each PR has its own file in this folder with a copy-paste prompt and a "Definition of done" checklist.

| PR | File | What it does | Risk |
|---|---|---|---|
| 1 | `pr-01-tokens.md` | Update `tokens.css` color and font variables. ~80% of the visual change lands here. | Low |
| 2 | `pr-02-fonts-wordmark.md` | Swap fonts (Bricolage → Lato), update nav wordmark to `STORA`, update favicon. | Low |
| 3 | `pr-03-components.md` | Refine `Button`, `Field`, `Masthead`, `Nav` against the spec. | Medium |
| 4 | `pr-04-audit.md` | Walk every page, fix dark-mode / error / focus edge cases. | Medium |

Between each PR: run the app, click through every route, eyeball it, commit. **Do not merge PR N+1 before PR N is on `main`.**

## What's in this bundle

```
design_handoff_stora_design_system/
├── README.md                       ← this file
├── pr-01-tokens.md                 ← PR 1 prompt + checklist
├── pr-02-fonts-wordmark.md         ← PR 2 prompt + checklist
├── pr-03-components.md             ← PR 3 prompt + checklist
├── pr-04-audit.md                  ← PR 4 prompt + checklist
└── design-system/
    ├── DESIGN_SYSTEM.md            ← Full design rules (voice, color, type, motion, components)
    ├── SKILL.md                    ← Claude Code skill manifest
    ├── colors_and_type.css         ← All tokens as plain CSS — reference
    └── assets/
        ├── favicon.svg             ← Inked S monogram
        └── icons.svg               ← Social/utility sprite (unchanged from current repo)
```

## About the design files

The files in `design-system/` are **the source of truth**, not production code to copy. `colors_and_type.css` is a self-contained reference of every token; the production codebase already has a similar `tokens.css` using Tailwind v4 `@theme` mapping that should be updated to match, not replaced. `DESIGN_SYSTEM.md` is the spec — read it before each PR.

## Fidelity

**High-fidelity.** Every color is an exact OKLCH value, every font size is named, every motion duration is fixed. PRs should reproduce the spec values exactly — no eyeballing, no "approximately."

## Skill installation (one-time, before PR 1)

Drop `design-system/SKILL.md` into your repo at `.claude/skills/stora-design.md` (Claude Code reads from `.claude/skills/`). Also copy `design-system/DESIGN_SYSTEM.md` to `docs/design-system/README.md` so it's discoverable from the repo root.

After that, every Claude Code session in this repo will have the design rules in context automatically — you don't need to paste the spec into each prompt.

## What's intentionally out of scope

The GitHub additions (cart panel slide-out, editorial product grid, admin chrome) **exist in the codebase but are not part of this design system** at the user's request. Do not introduce them. Do not remove them either — the PRs below touch tokens and chrome, not page-level layouts. If you later decide to fold them into the system, that's a separate design pass.

## Quick-reference design tokens

See `design-system/colors_and_type.css` for the full list. The most-important values:

| Token | Light | Dark | Use |
|---|---|---|---|
| `--surface-0` | `oklch(0.985 0.003 60)` | `oklch(0.14 0.008 250)` | Page bg |
| `--surface-1` | `oklch(1.00 0 0)` | `oklch(0.18 0.008 250)` | Cards |
| `--ink` | `oklch(0.18 0.01 25)` | `oklch(0.94 0.005 60)` | Body text |
| `--accent` | `oklch(0.62 0.17 38)` | `oklch(0.70 0.16 38)` | Buttons, focus, errors |
| `--highlight` | `oklch(0.88 0.13 85)` | `oklch(0.78 0.13 85)` | Sale/best-seller badges (new) |
| `--font-display` | Lato 700/900 | — | Headings, wordmark |
| `--font-body` | Hanken Grotesk variable | — | Everything else |

## After all four PRs land

The codebase should match every spec in `DESIGN_SYSTEM.md`. Final acceptance check: open the storefront, log in, walk Shop → Cart → Checkout → Pay → Order → Account → Admin, in both light and dark mode. Nothing should look "off-brand." If something does, file it as a follow-up — don't try to fix it in the audit PR if it's a new pattern.
