import { describe, it, expect } from "vitest";
import { hueFromString } from "./color";

describe("hueFromString", () => {
  it("is deterministic for the same input", () => {
    expect(hueFromString("payments-api")).toBe(hueFromString("payments-api"));
  });

  it("returns a hue in the 0–359 range", () => {
    for (const s of ["", "a", "payments-api", "a-very-long-project-slug-name", "Z9_x"]) {
      const h = hueFromString(s);
      expect(h).toBeGreaterThanOrEqual(0);
      expect(h).toBeLessThanOrEqual(359);
    }
  });

  it("maps the empty string to 0", () => {
    expect(hueFromString("")).toBe(0);
  });

  it("spreads different slugs across distinct hues", () => {
    const hues = new Set(["alpha", "beta", "gamma", "delta", "epsilon"].map(hueFromString));
    expect(hues.size).toBeGreaterThan(1);
  });
});
