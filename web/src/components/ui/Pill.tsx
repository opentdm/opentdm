import { ReactNode } from "react";
import { Label } from "../../ui/primer";

// A solid accent (iris) pill — the active "base" env chip and the "current"
// version marker. The .otdm-pill-accent class forces the accent-emphasis fill
// (see primitives.css), so no Primer variant is needed.
export function Pill({ children }: { children: ReactNode }) {
  return <Label className="otdm-pill-accent">{children}</Label>;
}
