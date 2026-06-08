// resolve.ts — the base⊕env merge engine, ported from the redesign prototype's
// engine.js and typed against the API. Pure (no I/O): callers fetch a config's
// base + env items and feed them in. Shared by the inherited-aware KV editor and
// the file-browser's read-only single view + delta badge.

import { Item } from "../api";

// Row inheritance state when an env layer overlays base:
//   base       — an own row of the base layer (key editable)
//   inherited  — tracks a base key, no env row written (rendered greyed)
//   override   — an env row that diverges from base (or an edited inherited key)
//   new        — an env-only key with no base default
//   tombstone  — a deleted=true env row that unsets an inherited base key
export type RowState = "base" | "inherited" | "override" | "new" | "tombstone";

export interface MergedRow {
  key: string;
  value: string;
  is_secret: boolean;
  baseValue?: string; // base default when base defines this key
  baseSecret?: boolean;
  state: RowState;
}

// buildRows merges a config's base layer with one env layer into sorted rows
// carrying inheritance state. isBase=true yields the base layer's own rows.
export function buildRows(baseItems: Item[], layerItems: Item[], isBase: boolean): MergedRow[] {
  if (isBase) {
    return baseItems
      .filter((i) => !i.deleted)
      .map((i) => ({ key: i.key, value: i.value, is_secret: i.is_secret, state: "base" as const }))
      .sort((a, b) => a.key.localeCompare(b.key));
  }
  const bMap = new Map(baseItems.filter((i) => !i.deleted).map((i) => [i.key, i] as const));
  const lMap = new Map(layerItems.map((i) => [i.key, i] as const));
  const keys = new Set<string>([...bMap.keys(), ...layerItems.map((i) => i.key)]);
  const rows: MergedRow[] = [];
  for (const key of keys) {
    const b = bMap.get(key);
    const l = lMap.get(key);
    if (l && l.deleted) {
      if (b) rows.push({ key, value: "", is_secret: false, baseValue: b.value, baseSecret: b.is_secret, state: "tombstone" });
      continue; // a tombstone with no base key is meaningless
    }
    if (l) {
      rows.push(
        b
          ? { key, value: l.value, is_secret: l.is_secret, baseValue: b.value, baseSecret: b.is_secret, state: "override" }
          : { key, value: l.value, is_secret: l.is_secret, state: "new" },
      );
    } else if (b) {
      rows.push({ key, value: b.value, is_secret: b.is_secret, baseValue: b.value, baseSecret: b.is_secret, state: "inherited" });
    }
  }
  rows.sort((a, b) => a.key.localeCompare(b.key));
  return rows;
}

export type Origin = "base" | "override" | "new";

export interface ResolvedEntry {
  key: string;
  value: string;
  is_secret: boolean;
  origin: Origin;
}

// resolvedEntries is the effective env set (tombstones dropped) for the read-only
// single view, with each key's origin for syntax tinting.
export function resolvedEntries(rows: MergedRow[]): ResolvedEntry[] {
  return rows
    .filter((r) => r.state !== "tombstone")
    .map((r) => ({
      key: r.key,
      value: r.value,
      is_secret: r.is_secret,
      origin: r.state === "override" ? "override" : r.state === "new" ? "new" : "base",
    }));
}

export interface EnvDelta {
  override: number;
  new: number;
  unset: number;
}

// envDelta counts how an env layer diverges from base (for the "N vs base" badge).
export function envDelta(rows: MergedRow[]): EnvDelta {
  return rows.reduce<EnvDelta>(
    (a, r) => {
      if (r.state === "override") a.override++;
      else if (r.state === "new") a.new++;
      else if (r.state === "tombstone") a.unset++;
      return a;
    },
    { override: 0, new: 0, unset: 0 },
  );
}
