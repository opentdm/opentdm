// App-wide color mode (light / dark / auto). Backed by the server-synced
// preferences store (see lib/preferences); this module just wires it to Primer's
// ThemeProvider and exposes the existing useColorMode() API.
import { ReactNode } from "react";
import { ThemeProvider } from "../ui/primer";
import { ColorMode, setColorMode, usePreferences } from "./preferences";

export type { ColorMode };

export function ColorModeProvider({ children }: { children: ReactNode }) {
  const { colorMode } = usePreferences();
  return <ThemeProvider colorMode={colorMode}>{children}</ThemeProvider>;
}

export function useColorMode(): { mode: ColorMode; setMode: (m: ColorMode) => void } {
  const { colorMode } = usePreferences();
  return { mode: colorMode, setMode: setColorMode };
}
