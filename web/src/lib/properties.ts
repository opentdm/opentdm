import { Item } from "../api";
import { ENV_KEY_RE } from "./dotenv";

const SECRET_HINT = /KEY|SECRET|TOKEN|PASS|DSN/i;

export interface ParsedProperties {
  items: Item[];
  invalidKeys: string[];
}

// Minimal Java .properties parser: '#'/'!' comments and '='/':'/whitespace
// separators. Keys are validated against the env grammar — properties keys with
// dots/dashes (a.b.c) can't be stored as env variables (the inject-safe renderer
// rejects them), so they're reported in invalidKeys for the caller to surface.
export function parseProperties(text: string): ParsedProperties {
  const items: Item[] = [];
  const invalidKeys: string[] = [];
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#") || line.startsWith("!")) continue;
    const sep = line.match(/[=:\s]/);
    let key: string;
    let value: string;
    if (sep && sep.index !== undefined && sep.index > 0) {
      key = line.slice(0, sep.index).trim();
      value = line.slice(sep.index + 1).trim();
    } else {
      key = line;
      value = "";
    }
    if (!ENV_KEY_RE.test(key) || key.startsWith("BASH_FUNC_")) {
      invalidKeys.push(key);
      continue;
    }
    items.push({ key, value, is_secret: SECRET_HINT.test(key), deleted: false });
  }
  return { items, invalidKeys };
}
