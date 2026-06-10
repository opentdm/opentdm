---
name: opentdm-design-tokens
description: opentdm's web color + theming system. Use when editing web/src/ui/tokens.css or primitives.css, adding or restyling any UI surface, choosing colors, or touching light/dark/auto theming. Covers the iris OKLCH accent (hue 278), the "only recolor the accent family" rule, the Primer functional-variable contract, the primary→iris button remap, the light/dark/auto selector triad, and per-entity hues.
---

# opentdm design tokens (color + theming)

The visual identity is defined in `web/src/ui/tokens.css` (imported AFTER Primer's theme CSS so equal-specificity overrides win) and `web/src/ui/primitives.css`. Fonts: Hanken Grotesk (UI) + JetBrains Mono (code). Accent: **iris, OKLCH hue 278**.

## Cardinal rule: only the accent family + neutral palette are overridden
`tokens.css` recolors ONLY the accent var family and the warm-neutral palette (`--fgColor-default/muted`, `--bgColor-default/muted/inset`, `--borderColor-*`, all hue 278). Everything else (success/attention/danger/done) stays on Primer's defaults. Don't recolor semantic families to "make it match" — use the accent or a neutral.

- Iris accent: light `--fgColor-accent: oklch(0.5 0.18 278)`, `--bgColor-accent-emphasis: oklch(0.55 0.17 278)`; dark `--fgColor-accent: oklch(0.78 0.13 278)`, `--bgColor-accent-emphasis: oklch(0.68 0.15 278)`.
- **Primary buttons are remapped to iris.** Primer's default `--button-primary-bgColor-rest` is green (`--bgColor-success-emphasis`); `tokens.css` overrides `--button-primary-bgColor-rest/-hover/-active` and `-borderColor-*` to the accent. So `<Button variant="primary">` is iris — use it for primary actions. (A green primary button means this override regressed.)

## Consume functional variables — never hardcode colors
In components and scoped CSS, reference Primer functional vars; do NOT write hex/rgb:
- text `var(--fgColor-default|muted|accent)`, surfaces `var(--bgColor-default|muted|inset|accent-muted)`, borders `var(--borderColor-default|muted|emphasis)`.
- In `sx`, use the token paths the shim maps (`color: "fg.muted"`, `bg: "canvas.subtle"`).
- Raw hex/oklch literals belong ONLY in `tokens.css`, `primitives.css`, and the Appearance preview swatches (`AppearancePanel.tsx`, which intentionally previews each theme regardless of the active one).
- Iris solid chip = `<Label className="otdm-pill-accent">`.

## Light / dark / auto — the selector triad
Theme is server-synced (`lib/preferences.ts`; server wins on hydrate) and applied by Primer's `ThemeProvider`. Any rule that hardcodes an OKLCH tint (e.g. `.otdm-avatar`, `.otdm-proj-ico`) must cover **all three** cases or it breaks in explicit-dark or auto-on-a-dark-system:
```css
.x { /* light */ }
[data-color-mode="dark"][data-dark-theme="dark"] .x,
[data-color-mode="auto"][data-light-theme="dark"] .x { /* dark */ }
@media (prefers-color-scheme: dark) {
  [data-color-mode="auto"][data-dark-theme="dark"] .x { /* dark (auto on dark system) */ }
}
```
Functional-var-based rules are theme-aware automatically and don't need this — only literal OKLCH/hex tints do.

## Per-entity tints
No color field exists on the API, so per-project / per-user colors derive a stable hue from a string: `hueFromString(slug|name)` (`web/src/lib/color.ts`) → an inline CSS `--h`, consumed by `.otdm-avatar` / `.otdm-proj-ico` as `oklch(L C var(--h))`. Reuse this for any new per-entity tint.

## After token/color changes
Rebuild the embed (Node 22) and eyeball **both** light and dark — `cd web && npm run build`. Confirm primary buttons are iris, secrets still mask, and no surface lost its dark variant.
