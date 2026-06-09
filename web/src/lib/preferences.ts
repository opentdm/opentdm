// Per-user UI preferences (theme + favourite project slugs), now server-backed.
// localStorage is kept as an instant cache so there's no first-paint flash; the
// server value (from /auth/me) is hydrated in once known and becomes the source
// of truth, and every change is persisted back via PUT /auth/me/preferences.
import { useMemo, useSyncExternalStore } from "react";
import { api } from "../api";

export type ColorMode = "light" | "dark" | "auto";
export interface Preferences {
  colorMode: ColorMode;
  favourites: string[];
}

const LS_KEY = "otdm-prefs";
const VALID_MODES: readonly ColorMode[] = ["light", "dark", "auto"];

function normalize(p: { colorMode?: unknown; favourites?: unknown }): Preferences {
  const colorMode = VALID_MODES.includes(p?.colorMode as ColorMode) ? (p.colorMode as ColorMode) : "auto";
  const favourites = Array.isArray(p?.favourites) ? (p.favourites as unknown[]).filter((x): x is string => typeof x === "string") : [];
  return { colorMode, favourites };
}

// One-time migration from the previous per-feature localStorage keys.
function migrateLegacy(): Preferences {
  let colorMode: ColorMode = "auto";
  let favourites: string[] = [];
  try {
    const m = localStorage.getItem("otdm-color-mode");
    if (m && VALID_MODES.includes(m as ColorMode)) colorMode = m as ColorMode;
    const f = localStorage.getItem("otdm-favs");
    if (f) favourites = JSON.parse(f) as string[];
  } catch {
    /* ignore */
  }
  return { colorMode, favourites };
}

function readLocal(): Preferences {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (raw) return normalize(JSON.parse(raw));
  } catch {
    /* ignore */
  }
  return migrateLegacy();
}

let current = readLocal();
let hydrated = false;
let saveTimer: number | undefined;
const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}
function subscribe(cb: () => void): () => void {
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

function persistLocal() {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(current));
  } catch {
    /* storage may be unavailable */
  }
}

// Debounced, fire-and-forget server save. Skipped until hydration so we never
// clobber server state with local defaults before we've read it.
function persistServer() {
  if (!hydrated) return;
  if (saveTimer) window.clearTimeout(saveTimer);
  saveTimer = window.setTimeout(() => {
    void api.put("/auth/me/preferences", { color_mode: current.colorMode, favourites: current.favourites }).catch(() => {});
  }, 400);
}

// hydratePreferences reconciles with /auth/me. If the server has values, they
// win (and update the cache); if it has none, we push our migrated local state
// up so it follows the account from here on.
export function hydratePreferences(serverPrefs: { color_mode?: string; favourites?: string[] } | undefined): void {
  const hasServer = !!serverPrefs && (!!serverPrefs.color_mode || (serverPrefs.favourites?.length ?? 0) > 0);
  if (hasServer) {
    current = normalize({ colorMode: serverPrefs!.color_mode, favourites: serverPrefs!.favourites });
    persistLocal();
    emit();
    hydrated = true;
  } else {
    hydrated = true;
    persistServer();
  }
}

export function setColorMode(mode: ColorMode): void {
  current = { ...current, colorMode: mode };
  persistLocal();
  persistServer();
  emit();
}

export function toggleFavourite(slug: string): void {
  const has = current.favourites.includes(slug);
  current = {
    ...current,
    favourites: has ? current.favourites.filter((s) => s !== slug) : [...current.favourites, slug],
  };
  persistLocal();
  persistServer();
  emit();
}

function getSnapshot(): Preferences {
  return current;
}

export function usePreferences(): Preferences {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

// Convenience selector: favourites as a Set (memoized on the stable array).
export function useFavouriteSet(): Set<string> {
  const { favourites } = usePreferences();
  return useMemo(() => new Set(favourites), [favourites]);
}
