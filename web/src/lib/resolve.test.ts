import { describe, it, expect } from "vitest";
import { Item } from "../api";
import { buildRows, resolvedEntries, envDelta, buildResolvedMap, diffMaps, diffLines } from "./resolve";

const item = (key: string, value: string, extra: Partial<Item> = {}): Item => ({
  key,
  value,
  is_secret: false,
  deleted: false,
  ...extra,
});

describe("buildRows", () => {
  it("isBase yields the base layer's own rows, sorted, with deleted dropped", () => {
    const base = [item("B", "2"), item("A", "1"), item("Z", "9", { deleted: true })];
    const rows = buildRows(base, [], true);
    expect(rows.map((r) => r.key)).toEqual(["A", "B"]);
    expect(rows.every((r) => r.state === "base")).toBe(true);
  });

  it("classifies inherited / override / new / tombstone", () => {
    const base = [item("KEEP", "base"), item("OVER", "base"), item("GONE", "base")];
    const layer = [
      item("OVER", "env"), // diverges from base -> override
      item("NEW", "envonly"), // no base default -> new
      item("GONE", "", { deleted: true }), // tombstone unsets inherited base key
    ];
    const rows = buildRows(base, layer, false);
    const byKey = Object.fromEntries(rows.map((r) => [r.key, r]));

    expect(byKey.KEEP.state).toBe("inherited");
    expect(byKey.KEEP.value).toBe("base");
    expect(byKey.OVER.state).toBe("override");
    expect(byKey.OVER.value).toBe("env");
    expect(byKey.OVER.baseValue).toBe("base");
    expect(byKey.NEW.state).toBe("new");
    expect(byKey.NEW.baseValue).toBeUndefined();
    expect(byKey.GONE.state).toBe("tombstone");
    expect(byKey.GONE.baseValue).toBe("base");
  });

  it("drops a tombstone that has no base key (meaningless)", () => {
    const rows = buildRows([], [item("X", "", { deleted: true })], false);
    expect(rows).toEqual([]);
  });

  it("carries baseSecret and is_secret through overrides", () => {
    const base = [item("S", "b", { is_secret: true })];
    const layer = [item("S", "e", { is_secret: false })];
    const [row] = buildRows(base, layer, false);
    expect(row.state).toBe("override");
    expect(row.is_secret).toBe(false);
    expect(row.baseSecret).toBe(true);
  });

  it("sorts merged rows by key", () => {
    const base = [item("b", "1"), item("a", "1")];
    const rows = buildRows(base, [item("c", "1")], false);
    expect(rows.map((r) => r.key)).toEqual(["a", "b", "c"]);
  });
});

describe("resolvedEntries", () => {
  it("drops tombstones and maps state to origin", () => {
    const base = [item("KEEP", "b"), item("OVER", "b"), item("GONE", "b")];
    const layer = [item("OVER", "e"), item("NEW", "n"), item("GONE", "", { deleted: true })];
    const entries = resolvedEntries(buildRows(base, layer, false));
    const byKey = Object.fromEntries(entries.map((e) => [e.key, e]));

    expect(byKey.GONE).toBeUndefined();
    expect(byKey.KEEP.origin).toBe("base");
    expect(byKey.OVER.origin).toBe("override");
    expect(byKey.NEW.origin).toBe("new");
  });
});

describe("envDelta", () => {
  it("counts override / new / unset", () => {
    const base = [item("KEEP", "b"), item("OVER", "b"), item("GONE", "b")];
    const layer = [item("OVER", "e"), item("NEW", "n"), item("GONE", "", { deleted: true })];
    expect(envDelta(buildRows(base, layer, false))).toEqual({ override: 1, new: 1, unset: 1 });
  });

  it("is all zeros when an env layer is empty (pure inheritance)", () => {
    const base = [item("A", "1"), item("B", "2")];
    expect(envDelta(buildRows(base, [], false))).toEqual({ override: 0, new: 0, unset: 0 });
  });
});

describe("diffMaps", () => {
  it("returns the sorted union of keys and flags differing + absent keys", () => {
    const a = buildResolvedMap(buildRows([item("SAME", "x"), item("DIFF", "1"), item("ONLYA", "a")], [], true));
    const b = buildResolvedMap(buildRows([item("SAME", "x"), item("DIFF", "2")], [], true));
    const { keys, diff } = diffMaps([a, b]);
    expect(keys).toEqual(["DIFF", "ONLYA", "SAME"]);
    expect(diff.has("DIFF")).toBe(true); // value differs
    expect(diff.has("ONLYA")).toBe(true); // absent from pane b
    expect(diff.has("SAME")).toBe(false);
  });
});

describe("diffLines", () => {
  it("aligns panes by line index and flags differing + short lines", () => {
    const { max, diff } = diffLines([
      ["a", "b", "c"],
      ["a", "X", "c", "d"],
    ]);
    expect(max).toBe(4);
    expect(diff.has(0)).toBe(false);
    expect(diff.has(1)).toBe(true); // b vs X
    expect(diff.has(2)).toBe(false);
    expect(diff.has(3)).toBe(true); // present only in the second pane
  });
});
