import { Item } from "../api";

// Env variable name grammar — mirrors the server's codec.ValidKey, which
// re-validates every key on PUT (this is a friendly client-side pre-check).
export const ENV_KEY_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

// Keys whose name hints at a secret get flagged is_secret on import.
const SECRET_HINT = /KEY|SECRET|TOKEN|PASS|DSN/i;

export interface ParsedEnv {
  items: Item[];
  invalidKeys: string[];
}

// parseDotenv turns a .env document into base-layer items, inferring is_secret
// from the key name. It accepts `KEY=value` and `export KEY=value`, `#`
// comments, blank lines, CRLF, and single/double-quoted values. Lines without
// `=` are skipped; keys failing the grammar are collected in invalidKeys so the
// caller can surface them instead of silently dropping data.
export function parseDotenv(text: string): ParsedEnv {
  const items: Item[] = [];
  const invalidKeys: string[] = [];
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const body = line.startsWith("export ") ? line.slice(7).trim() : line;
    const eq = body.indexOf("=");
    if (eq < 0) continue;
    const key = body.slice(0, eq).trim();
    let value = body.slice(eq + 1).trim();
    if (
      value.length >= 2 &&
      ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'")))
    ) {
      value = value.slice(1, -1);
    }
    if (!ENV_KEY_RE.test(key) || key.startsWith("BASH_FUNC_")) {
      invalidKeys.push(key);
      continue;
    }
    items.push({ key, value, is_secret: SECRET_HINT.test(key), deleted: false });
  }
  return { items, invalidKeys };
}
