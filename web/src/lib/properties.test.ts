import { describe, it, expect } from "vitest";
import { parseProperties } from "./properties";

describe("parseProperties", () => {
  it("accepts '=', ':' and whitespace separators", () => {
    const { items, invalidKeys } = parseProperties(["A=1", "B:2", "C 3"].join("\n"));
    expect(invalidKeys).toEqual([]);
    expect(items.map((i) => [i.key, i.value])).toEqual([
      ["A", "1"],
      ["B", "2"],
      ["C", "3"],
    ]);
  });

  it("skips '#' and '!' comments and blank lines", () => {
    const { items } = parseProperties(["# comment", "! also comment", "", "KEY=v"].join("\n"));
    expect(items.map((i) => i.key)).toEqual(["KEY"]);
  });

  it("rejects dotted/dashed keys (not env-storable) into invalidKeys", () => {
    const { items, invalidKeys } = parseProperties(["a.b.c=1", "x-y=2", "OK=3"].join("\n"));
    expect(items.map((i) => i.key)).toEqual(["OK"]);
    expect(invalidKeys).toEqual(["a.b.c", "x-y"]);
  });

  it("treats a key with no separator as an empty value", () => {
    const { items } = parseProperties("LONELY");
    expect(items).toEqual([{ key: "LONELY", value: "", is_secret: false, deleted: false }]);
  });

  it("flags secret-looking keys is_secret", () => {
    const { items } = parseProperties(["SECRET_TOKEN=t", "PLAIN=v"].join("\n"));
    const byKey = Object.fromEntries(items.map((i) => [i.key, i.is_secret]));
    expect(byKey.SECRET_TOKEN).toBe(true);
    expect(byKey.PLAIN).toBe(false);
  });
});
