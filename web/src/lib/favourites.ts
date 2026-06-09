// Favourite (pinned) projects. Backed by the server-synced preferences store
// (see lib/preferences). Kept as a thin shim so existing call sites are unchanged.
import { toggleFavourite, useFavouriteSet } from "./preferences";

export function useFavourites(): { favs: Set<string>; toggle: (slug: string) => void } {
  return { favs: useFavouriteSet(), toggle: toggleFavourite };
}
