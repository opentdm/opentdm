// Primer 38 compatibility layer.
//
// Primer v37+ removed the styled-system `sx` prop and the `Box` component. This
// module restores both with a faithful, static `sx` → inline-style mapping so the
// ~220 existing call sites keep working unchanged: every file imports its Primer
// components from here instead of "@primer/react". It re-exports everything else
// verbatim and overrides only the components used with `sx`.
//
// The app's `sx` is entirely static (no pseudo-selectors / responsive arrays —
// verified), so a `style`-based shim is exact. Theme tokens resolve to Primer 38
// functional CSS variables (@primer/primitives v11).
import React from "react";
import {
  Text as PrimerText,
  Heading as PrimerHeading,
  Flash as PrimerFlash,
  Label as PrimerLabel,
  FormControl as PrimerFormControl,
  Spinner as PrimerSpinner,
  TextInput as PrimerTextInput,
  Textarea as PrimerTextarea,
} from "@primer/react";

// Re-export the full Primer surface; the named exports below shadow the ones
// that need `sx` support.
export * from "@primer/react";

export type Sx = Record<string, string | number | null | undefined>;

// Primer theme scales (px). Indexing an `sx` numeric value through these matches
// styled-system's behavior (e.g. mb:2 → 8px, fontSize:3 → 20px).
const SPACE = [0, 4, 8, 16, 24, 32, 40, 48, 64, 80, 96, 112, 128];
const FONT_SIZES = [12, 14, 16, 20, 24, 32, 40, 48];
const RADII = [0, 3, 6, 12];

function spaceVal(v: string | number): string | number {
  if (typeof v !== "number") return v;
  const px = SPACE[Math.abs(v)] ?? Math.abs(v);
  return v < 0 ? -px : px;
}

// Resolve a Primer color token to its Primer-38 functional CSS variable with the
// light-theme hex as a fallback. The fallback matters: the legacy
// ThemeProvider/BaseStyles path doesn't reliably emit these vars on its wrapper,
// so the hex guarantees correct light rendering now; the var() auto-upgrades to
// mode-aware if the variable is present. Hexes are exact @primer/primitives v11
// light values. Unknown tokens pass through as literal CSS.
const COLOR_TOKENS: Record<string, string> = {
  "fg.muted": "var(--fgColor-muted, #59636e)",
  "fg.default": "var(--fgColor-default, #1f2328)",
  "fg.onEmphasis": "var(--fgColor-onEmphasis, #ffffff)",
  "danger.fg": "var(--fgColor-danger, #d1242f)",
  "success.fg": "var(--fgColor-success, #1a7f37)",
  "border.default": "var(--borderColor-default, #d1d9e0)",
  "border.muted": "var(--borderColor-muted, #d1d9e0b3)",
  "canvas.subtle": "var(--bgColor-muted, #f6f8fa)",
  "canvas.default": "var(--bgColor-default, #ffffff)",
};
function colorVal(v: string): string {
  return COLOR_TOKENS[v] ?? v;
}

const MARGIN: Record<string, string> = { m: "margin", mt: "marginTop", mb: "marginBottom", ml: "marginLeft", mr: "marginRight" };
const PADDING: Record<string, string> = { p: "padding", pt: "paddingTop", pb: "paddingBottom", pl: "paddingLeft", pr: "paddingRight" };

export function sxToStyle(sx?: Sx): React.CSSProperties | undefined {
  if (!sx) return undefined;
  const s: Record<string, string | number> = {};
  for (const [k, raw] of Object.entries(sx)) {
    if (raw == null) continue;
    if (k in MARGIN) { s[MARGIN[k]] = spaceVal(raw); continue; }
    if (k in PADDING) { s[PADDING[k]] = spaceVal(raw); continue; }
    switch (k) {
      case "gap": s.gap = spaceVal(raw); break;
      case "mx": { const v = spaceVal(raw); s.marginLeft = v; s.marginRight = v; break; }
      case "my": { const v = spaceVal(raw); s.marginTop = v; s.marginBottom = v; break; }
      case "px": { const v = spaceVal(raw); s.paddingLeft = v; s.paddingRight = v; break; }
      case "py": { const v = spaceVal(raw); s.paddingTop = v; s.paddingBottom = v; break; }
      case "fontSize": s.fontSize = typeof raw === "number" ? (FONT_SIZES[raw] ?? raw) : raw; break;
      case "borderRadius": s.borderRadius = typeof raw === "number" ? (RADII[raw] ?? raw) : raw; break;
      case "color": s.color = colorVal(String(raw)); break;
      case "bg":
      case "backgroundColor": s.backgroundColor = colorVal(String(raw)); break;
      case "borderColor": s.borderColor = colorVal(String(raw)); break;
      case "borderBottomColor": s.borderBottomColor = colorVal(String(raw)); break;
      case "borderTopColor": s.borderTopColor = colorVal(String(raw)); break;
      case "fontFamily": s.fontFamily = raw === "mono" ? "var(--fontStack-monospace, ui-monospace, SFMono-Regular, monospace)" : String(raw); break;
      default: s[k] = raw; // display, flex*, align*, justify*, width, border*, overflow, cursor, … pass through
    }
  }
  return s as React.CSSProperties;
}

// withSx adds an `sx` prop to a Primer component, merging it into `style`
// (explicit `style` wins). Preserves the component's own prop types.
function withSx<C extends React.ElementType>(Comp: C) {
  type Props = React.ComponentProps<C> & { sx?: Sx };
  function SxWrapped({ sx, style, ...rest }: Props & { style?: React.CSSProperties }) {
    return React.createElement(Comp, {
      ...rest,
      style: { ...sxToStyle(sx), ...style },
    } as React.ComponentProps<C>);
  }
  SxWrapped.displayName = `withSx(${(Comp as { displayName?: string }).displayName ?? "Component"})`;
  return SxWrapped;
}

// Box: a styled-system-free replacement (Primer 38 removed it).
export interface BoxProps extends React.HTMLAttributes<HTMLElement> {
  as?: React.ElementType;
  sx?: Sx;
  htmlFor?: string;
  to?: string; // when as={RouterLink}
}
export const Box = React.forwardRef<HTMLElement, BoxProps>(function Box({ as: As = "div", sx, style, ...rest }, ref) {
  return React.createElement(As, { ref, style: { ...sxToStyle(sx), ...style }, ...rest });
});

export const Text = withSx(PrimerText);
export const Heading = withSx(PrimerHeading);
export const Flash = withSx(PrimerFlash);
// Link is NOT wrapped: it's used polymorphically (`as={RouterLink} to=…`), and
// the raw Primer Link preserves that; its one sx call site uses sxToStyle.
export const Label = withSx(PrimerLabel);
export const FormControl = Object.assign(withSx(PrimerFormControl), {
  Label: PrimerFormControl.Label,
  Caption: PrimerFormControl.Caption,
  Validation: PrimerFormControl.Validation,
  LeadingVisual: PrimerFormControl.LeadingVisual,
});
export const Spinner = withSx(PrimerSpinner);
// TextInput/Textarea forward `style` to their wrapper; fontFamily inherits to
// the input, and width/flex on the wrapper is the intended layout target.
export const TextInput = Object.assign(withSx(PrimerTextInput), { Action: PrimerTextInput.Action });
export const Textarea = withSx(PrimerTextarea);
