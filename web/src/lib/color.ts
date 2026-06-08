// Deterministic per-project tint. The API has no color field, so we derive a
// stable OKLCH hue (0–359) from the slug; CSS turns the hue into theme-aware
// fill/border (see .otdm-proj-ico in shell.css).
export function hueFromString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) % 360;
  }
  return h;
}
