// App-wide color mode (light / dark / auto), persisted to localStorage. Wraps
// Primer's ThemeProvider so the choice drives the whole UI; there is no
// server-side preference, so this is client-only (matches the redesign).
import { createContext, useCallback, useContext, useState, ReactNode } from "react";
import { ThemeProvider } from "../ui/primer";

export type ColorMode = "light" | "dark" | "auto";

const KEY = "otdm-color-mode";
const VALID: readonly ColorMode[] = ["light", "dark", "auto"];

function read(): ColorMode {
  try {
    const v = localStorage.getItem(KEY);
    return v && (VALID as readonly string[]).includes(v) ? (v as ColorMode) : "auto";
  } catch {
    return "auto";
  }
}

interface ColorModeCtx {
  mode: ColorMode;
  setMode: (m: ColorMode) => void;
}

const Ctx = createContext<ColorModeCtx>({ mode: "auto", setMode: () => {} });

export function ColorModeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ColorMode>(read);
  const setMode = useCallback((m: ColorMode) => {
    setModeState(m);
    try {
      localStorage.setItem(KEY, m);
    } catch {
      /* storage may be unavailable (private mode); keep in-memory state */
    }
  }, []);

  return (
    <Ctx.Provider value={{ mode, setMode }}>
      <ThemeProvider colorMode={mode}>{children}</ThemeProvider>
    </Ctx.Provider>
  );
}

export function useColorMode(): ColorModeCtx {
  return useContext(Ctx);
}
