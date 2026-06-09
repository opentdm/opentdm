import { ReactNode } from "react";

// Small iris uppercase eyebrow rendered above a page or section title (e.g.
// "PROJECT", "SETTINGS", "ACCOUNT"). See .otdm-overline in primitives.css.
export default function Overline({ children }: { children: ReactNode }) {
  return <span className="otdm-overline">{children}</span>;
}
