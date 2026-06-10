---
name: opentdm-web
description: Conventions for opentdm's React + Vite + Primer single-page app in web/. Use when editing web/src (components, pages, editors, the api client, ui/* styles) or rebuilding the embedded UI. Covers the Primer sx-shim rule, the hybrid Primer-plus-scoped-CSS styling pattern, the Node-22 embed build, reusable patterns (api.ts, EditorDispatch, lib/resolve.ts, filebrowser), and UI secret handling. For colors/theming see opentdm-design-tokens; for raw build/test commands see opentdm-testing.
---

# opentdm web (the SPA)

React 19 + Vite + `@primer/react`, built to a single bundle the Go server serves via `go:embed`. Source is `web/src`; the build output `server/internal/webui/dist` is **committed**. Break these and styling silently fails or the committed embed diverges from CI.

## Cardinal rule: import Primer from the shim, never `@primer/react`
Components import `Box`, `Text`, `Heading`, `Label`, `Flash`, `FormControl`, `Spinner`, `TextInput`, `Button`, etc. from the local shim:
```ts
import { Box, Button, Label } from "../ui/primer";      // pages/ and components/
import { Box } from "../../ui/primer";                   // components/editors|filebrowser|settings/
```
`web/src/ui/primer.tsx` re-exports `@primer/react` but restores `sx` (Primer 38 dropped it) by mapping it to inline `style`. A direct `@primer/react` import loses that mapping — the component renders unstyled. The ONLY file allowed to import `@primer/react` directly is `ui/primer.tsx` itself.

## `sx` must be STATIC
The shim maps `sx` → a plain `style` object. So `sx` may contain only static values:
- **No** responsive arrays (`sx={{ p: [2, 3] }}`), **no** pseudo-selectors or `&` (`sx={{ ":hover": … }}`), no media queries.
- For hover/active/focus, pseudo-elements, line-numbered code, syntax tint, grids, or diff highlights, use a `className` + a rule in `web/src/ui/*.css`.
- Color values in `sx` use Primer token paths the shim understands: `{ color: "fg.muted" | "fg.accent" | "fg.default" }`, `{ bg: "canvas.subtle" }`, `borderColor: "border.default"`.

## Hybrid styling
Primer components for chrome (buttons, menus, dialogs, labels); scoped CSS (`web/src/ui/*.css`, classNames `otdm-*`, imported in `main.tsx`) for everything `sx` can't express. **All scoped CSS uses Primer functional CSS variables** (`--fgColor-*`, `--bgColor-*`, `--borderColor-*`) so it tracks light/dark — never hardcode colors (see `opentdm-design-tokens`).

## The embed build (the #1 CI footgun)
- `cd web && npm run build` → writes `../server/internal/webui/dist` with **stable filenames** (`assets/app.js`, `assets/index.css`; no content hashes), `emptyOutDir`.
- **Build with Node 22** (`web/.nvmrc`). The dev host is Node 24; building on the wrong version (or with a stale `node_modules`) makes the committed embed differ from CI's, and the `Web build` job fails on `git diff --quiet server/internal/webui/dist`. If host ≠ Node 22, build in a container:
  `docker run --rm -v "$PWD":/app -v otdm_embed_nm:/app/web/node_modules -w /app/web node:22-alpine sh -c "npm ci && npm run build"`
- After ANY change under `web/src`, rebuild and commit `server/internal/webui/dist`. Run `cd web && npm run typecheck` too — `vite build` skips types, but CI enforces `tsc --noEmit`.
- **Never hand-edit `server/internal/webui/dist`** — it's generated (the `guard-generated.sh` hook blocks it).

## Reusable patterns (use these, don't reinvent)
- **`web/src/api.ts`** — the typed client. Responses unwrap a `{ data }` envelope; mutations send the `X-CSRF-Token` header from the `otdm_csrf` cookie (double-submit). Add new endpoints here, keeping interfaces in lockstep with the server DTOs.
- **`web/src/lib/resolve.ts`** — the **pure** base⊕env merge engine (no I/O): `buildRows(baseItems, layerItems, isBase)` → `MergedRow[]` with a `RowState` of base/inherited/override/new/tombstone; `resolvedEntries`, `envDelta`, `buildResolvedMap`, `diffMaps`/`diffLines` (split-compare). Reuse it for any merge/diff UI; keep it pure and unit-tested.
- **`web/src/components/editors/EditorDispatch.tsx`** — routes `config.format` to `KvEditor` (env/properties/secret), `CodeEditor` (json/xml/yaml), or `CsvEditor`. Editors take `onSaved` to refresh siblings.
- **`web/src/components/filebrowser/*`** — `FileTree`, `BranchEnvMenu` (branch-style env picker), `CodeFileView` (read-only line-numbered, masks secrets), `SplitCompare` (2–3 panes), `DeltaBadge`.
- **Shared primitives** — `components/Overline.tsx`, `components/Avatar.tsx` (per-name OKLCH tint via `lib/color.ts`); reuse rather than re-implementing eyebrows/avatars.

## Secrets in the UI
Secret values render masked (`••••••••`) unless the user toggles Reveal; the per-row Reveal control is `aria-label`led. Never put secret values in logs, copy-to-clipboard text by default, or version/diff views (the server masks them; keep the client honest).

## After web changes
```bash
cd web && npm run typecheck
cd web && npm run build          # Node 22; commit server/internal/webui/dist
```
