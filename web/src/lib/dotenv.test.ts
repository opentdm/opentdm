import { describe, it, expect } from "vitest";
import { parseDotenv } from "./dotenv";

describe("parseDotenv", () => {
  it("parses KEY=value, export, comments, and blank lines", () => {
    const { items, invalidKeys } = parseDotenv(["# a comment", "", "FOO=bar", "export BAZ=qux"].join("\n"));
    expect(invalidKeys).toEqual([]);
    expect(items.map((i) => [i.key, i.value])).toEqual([
      ["FOO", "bar"],
      ["BAZ", "qux"],
    ]);
  });

  it("strips matching single or double quotes and handles CRLF", () => {
    const { items } = parseDotenv("A=\"hello world\"\r\nB='single'\r\nC=plain");
    expect(items.map((i) => i.value)).toEqual(["hello world", "single", "plain"]);
  });

  it("skips lines without '=' and collects invalid keys", () => {
    const { items, invalidKeys } = parseDotenv(["NO_EQUALS_LINE", "1BAD=x", "BASH_FUNC_evil=y", "GOOD=z"].join("\n"));
    expect(items.map((i) => i.key)).toEqual(["GOOD"]);
    expect(invalidKeys).toEqual(["1BAD", "BASH_FUNC_evil"]);
  });

  it("flags secret-looking keys is_secret", () => {
    const { items } = parseDotenv(["API_KEY=k", "DB_PASSWORD=p", "PLAIN=v"].join("\n"));
    const byKey = Object.fromEntries(items.map((i) => [i.key, i.is_secret]));
    expect(byKey.API_KEY).toBe(true);
    expect(byKey.DB_PASSWORD).toBe(true);
    expect(byKey.PLAIN).toBe(false);
  });
});
