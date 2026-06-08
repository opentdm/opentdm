// Per-user "pinned" projects. Stored in localStorage (no backend preference
// endpoint yet); exposed as an external store so the sidebar and the Projects
// grid stay in sync when a star is toggled in either place.
import { useSyncExternalStore } from "react";

const KEY = "otdm-favs";

function read(): Set<string> {
  try {
    const raw = localStorage.getItem(KEY);
    return new Set(raw ? (JSON.parse(raw) as string[]) : []);
  } catch {
    return new Set();
  }
}

let current = read();
const listeners = new Set<() => void>();

function emit(): void {
  for (const l of listeners) l();
}

export function getFavourites(): Set<string> {
  return current;
}

export function toggleFavourite(slug: string): void {
  const next = new Set(current);
  if (next.has(slug)) next.delete(slug);
  else next.add(slug);
  current = next;
  try {
    localStorage.setItem(KEY, JSON.stringify([...next]));
  } catch {
    /* storage may be unavailable (private mode); keep in-memory state */
  }
  emit();
}

function subscribe(cb: () => void): () => void {
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

// useFavourites returns the live set plus a toggle. getFavourites returns a
// stable reference between toggles, so useSyncExternalStore won't loop.
export function useFavourites(): { favs: Set<string>; toggle: (slug: string) => void } {
  const favs = useSyncExternalStore(subscribe, getFavourites, getFavourites);
  return { favs, toggle: toggleFavourite };
}
