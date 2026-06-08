import { ChangeEvent, useEffect, useRef, useState } from "react";
import { Box, Button, Flash, IconButton, Label, Spinner, Text, TextInput, Textarea } from "../../ui/primer";
import { EyeClosedIcon, EyeIcon, LockIcon, TrashIcon } from "@primer/octicons-react";
import { api, Config, Item } from "../../api";

const keyRe = /^[A-Za-z_][A-Za-z0-9_]*$/;

// Row state machine for the inherited-aware editor (non-base layers overlay base):
//   base       — an own row of the base layer (key editable)
//   inherited  — tracking a base key, no layer row will be written (greyed)
//   override   — a layer row that diverges from base (or an edited inherited key)
//   new        — a layer-only key with no base default (key editable)
//   tombstone  — a deleted=true layer row that unsets an inherited base key
type RowState = "base" | "inherited" | "override" | "new" | "tombstone";

interface Row {
  key: string;
  value: string;
  is_secret: boolean;
  reveal: boolean;
  baseValue?: string; // base default when base defines this key
  baseSecret?: boolean;
  state: RowState;
}

interface BaseVal {
  value: string;
  is_secret: boolean;
}

interface KvEditorProps {
  slug: string;
  config: Config;
  layer: string;
  readOnly?: boolean;
  onSaved?: () => void;
}

export default function KvEditor({ slug, config, layer, readOnly, onSaved }: KvEditorProps) {
  const isBase = layer === "base";
  const [rows, setRows] = useState<Row[]>([]);
  const [baseMap, setBaseMap] = useState<Map<string, BaseVal>>(new Map());
  const [raw, setRaw] = useState(false);
  const [rawText, setRawText] = useState("");
  const [loading, setLoading] = useState(true);
  const [reloadNonce, setReloadNonce] = useState(0);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMsg("");
    setErr("");
    setRaw(false);
    setLoading(true);
    const load = async () => {
      if (isBase) {
        const items = await api.getItems(slug, config.id, "base");
        setBaseMap(new Map());
        setRows(
          items
            .filter((i) => !i.deleted)
            .map((i) => ({ key: i.key, value: i.value, is_secret: i.is_secret, reveal: false, state: "base" as const })),
        );
        return;
      }
      const [baseItems, layerItems] = await Promise.all([
        api.getItems(slug, config.id, "base"),
        api.getItems(slug, config.id, layer),
      ]);
      const bMap = new Map<string, BaseVal>(
        baseItems.filter((i) => !i.deleted).map((i) => [i.key, { value: i.value, is_secret: i.is_secret }]),
      );
      setBaseMap(bMap);
      setRows(buildRows(bMap, layerItems));
    };
    load()
      .catch((e: any) => setErr(e.message))
      .finally(() => setLoading(false));
  }, [slug, config.id, layer, reloadNonce]);

  function update(i: number, patch: Partial<Row>) {
    setRows((rs) => rs.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }
  // Editing an inherited (or restoring a tombstoned) key turns it into an override.
  function editValue(i: number, value: string) {
    setRows((rs) =>
      rs.map((r, idx) => (idx === i ? { ...r, value, state: r.state === "inherited" || r.state === "tombstone" ? "override" : r.state } : r)),
    );
  }
  function toggleSecret(i: number) {
    setRows((rs) =>
      rs.map((r, idx) => (idx === i ? { ...r, is_secret: !r.is_secret, state: r.state === "inherited" ? "override" : r.state } : r)),
    );
  }
  function removeRow(i: number) {
    setRows((rs) => {
      const r = rs[i];
      if (r.state === "inherited") {
        // Unset an inherited base key for this environment (tombstone).
        return rs.map((x, idx) => (idx === i ? { ...x, value: "", is_secret: false, state: "tombstone" } : x));
      }
      if (r.state === "override" && r.baseValue !== undefined) {
        // Revert an override back to tracking base.
        return rs.map((x, idx) =>
          idx === i ? { ...x, value: x.baseValue ?? "", is_secret: x.baseSecret ?? false, reveal: false, state: "inherited" } : x,
        );
      }
      // base / new / override-without-base: drop the row entirely.
      return rs.filter((_, idx) => idx !== i);
    });
  }
  function restoreRow(i: number) {
    setRows((rs) =>
      rs.map((x, idx) =>
        idx === i ? { ...x, value: x.baseValue ?? "", is_secret: x.baseSecret ?? false, reveal: false, state: "inherited" } : x,
      ),
    );
  }
  function addRow() {
    setRows((rs) => [
      ...rs,
      { key: "", value: "", is_secret: config.format === "secret", reveal: true, state: isBase ? "base" : "new" },
    ]);
  }

  function toRaw() {
    // Serialize the effective set (everything except tombstones) as KEY=value.
    setRawText(
      rows
        .filter((r) => r.state !== "tombstone")
        .map((r) => `${r.key}=${r.value}`)
        .join("\n"),
    );
    setRaw(true);
  }
  function fromRaw() {
    setRows(classify(parseRawLines(rawText), baseMap, secretMap(rows), isBase, config.format === "secret"));
    setRaw(false);
  }
  function importEnv(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (!f) return;
    f.text().then((t) => {
      setRawText((p) => (p ? p + "\n" : "") + t);
      setRaw(true);
    });
    e.target.value = "";
  }

  async function save() {
    setErr("");
    setMsg("");
    const data = raw ? classify(parseRawLines(rawText), baseMap, secretMap(rows), isBase, config.format === "secret") : rows;
    for (const r of data) {
      const willWrite = isBase || r.state === "override" || r.state === "new";
      if (willWrite && !keyRe.test(r.key)) {
        setErr(`Invalid key: ${r.key || "(empty)"} — must match [A-Za-z_][A-Za-z0-9_]*`);
        return;
      }
    }
    // Send only this layer's own rows: overrides + new keys + tombstones. Inherited
    // rows are omitted so they keep tracking base. The base layer sends everything.
    const payload: Item[] = data
      .filter((r) => (isBase ? r.key !== "" : r.state === "override" || r.state === "new" || r.state === "tombstone"))
      .map((r) =>
        r.state === "tombstone"
          ? { key: r.key, value: "", is_secret: false, deleted: true }
          : { key: r.key, value: r.value, is_secret: r.is_secret, deleted: false },
      );
    try {
      await api.putItems(slug, config.id, layer, payload);
      setMsg(`Saved ${layer}.`);
      setRaw(false);
      setReloadNonce((n) => n + 1); // refetch so origins/badges re-settle
      onSaved?.(); // let siblings (resolved view) refresh — base may have changed too
    } catch (e: any) {
      setErr(e.message);
    }
  }

  if (loading) return <Spinner />;
  return (
    <Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {raw ? (
        <Textarea block rows={10} value={rawText} onChange={(e) => setRawText(e.target.value)} sx={{ fontFamily: "mono" }} />
      ) : (
        <Box sx={{ display: "grid", gap: 1 }}>
          {rows.length === 0 && <Text sx={{ color: "fg.muted" }}>No variables{isBase ? "" : " for this environment"}.</Text>}
          {rows.map((r, i) =>
            r.state === "tombstone" ? (
              <Box key={i} sx={{ display: "flex", gap: 1, alignItems: "center" }}>
                <TextInput value={r.key} disabled sx={{ width: 240, fontFamily: "mono", color: "fg.muted" }} />
                <Text sx={{ flex: 1, color: "fg.muted", fontSize: 0 }}>
                  unset — hidden from <b>{layer}</b>
                </Text>
                <SourceBadge state={r.state} />
                {!readOnly && (
                  <Button size="small" onClick={() => restoreRow(i)}>
                    Restore
                  </Button>
                )}
              </Box>
            ) : (
              <Box key={i} sx={{ display: "flex", gap: 1, alignItems: "center" }}>
                <TextInput
                  value={r.key}
                  onChange={(e) => update(i, { key: e.target.value })}
                  placeholder="KEY"
                  disabled={readOnly || !(r.state === "base" || r.state === "new")}
                  sx={{ width: 240, fontFamily: "mono", color: r.state === "inherited" ? "fg.muted" : undefined }}
                />
                <TextInput
                  block
                  value={r.value}
                  onChange={(e) => editValue(i, e.target.value)}
                  placeholder="value"
                  type={r.is_secret && !r.reveal ? "password" : "text"}
                  disabled={readOnly}
                  sx={{ flex: 1, fontFamily: "mono", color: r.state === "inherited" ? "fg.muted" : undefined }}
                />
                <SourceBadge state={r.state} />
                {!readOnly && (
                  <IconButton
                    icon={LockIcon}
                    aria-label="mark secret"
                    size="small"
                    variant={r.is_secret ? "primary" : "default"}
                    onClick={() => toggleSecret(i)}
                  />
                )}
                {r.is_secret && (
                  <IconButton
                    icon={r.reveal ? EyeIcon : EyeClosedIcon}
                    aria-label="reveal value"
                    size="small"
                    onClick={() => update(i, { reveal: !r.reveal })}
                  />
                )}
                {!readOnly && (
                  <IconButton
                    icon={TrashIcon}
                    aria-label={r.state === "inherited" ? "unset for this environment" : "delete"}
                    size="small"
                    variant="danger"
                    onClick={() => removeRow(i)}
                  />
                )}
              </Box>
            ),
          )}
        </Box>
      )}
      {!readOnly && (
        <Box sx={{ mt: 2, display: "flex", gap: 2, alignItems: "center", flexWrap: "wrap" }}>
          <Button variant="primary" onClick={save}>
            Save {layer}
          </Button>
          {!raw && <Button onClick={addRow}>Add variable</Button>}
          <Button onClick={() => (raw ? fromRaw() : toRaw())}>{raw ? "Table view" : "Raw .env"}</Button>
          <Button onClick={() => fileInput.current?.click()}>Import .env</Button>
          <input ref={fileInput} type="file" accept=".env,text/plain" hidden onChange={importEnv} />
          {msg && <Text sx={{ color: "success.fg" }}>{msg}</Text>}
        </Box>
      )}
      <Text sx={{ color: "fg.muted", fontSize: 0, display: "block", mt: 2 }}>
        {isBase ? (
          <>
            Editing the <b>base</b> layer — these defaults are inherited by every environment.
          </>
        ) : (
          <>
            Greyed rows are inherited from <b>base</b>. Edit one to override it for <b>{layer}</b>; delete one to unset it
            here.
          </>
        )}
      </Text>
    </Box>
  );
}

// SourceBadge labels a row's origin. Base-layer own rows get no badge (every key
// is base-defined there).
function SourceBadge({ state }: { state: RowState }) {
  switch (state) {
    case "inherited":
      return <Label variant="secondary">base</Label>;
    case "override":
      return <Label variant="accent">override</Label>;
    case "new":
      return <Label variant="success">new</Label>;
    case "tombstone":
      return <Label variant="danger">unset</Label>;
    default:
      return null;
  }
}

function buildRows(baseMap: Map<string, BaseVal>, layerItems: Item[]): Row[] {
  const lMap = new Map(layerItems.map((i) => [i.key, i]));
  const keys = new Set<string>([...baseMap.keys(), ...layerItems.map((i) => i.key)]);
  const rows: Row[] = [];
  for (const key of keys) {
    const b = baseMap.get(key);
    const l = lMap.get(key);
    if (l && l.deleted) {
      if (b) rows.push({ key, value: "", is_secret: false, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "tombstone" });
      continue; // a tombstone with no base key is meaningless
    }
    if (l) {
      rows.push(
        b
          ? { key, value: l.value, is_secret: l.is_secret, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "override" }
          : { key, value: l.value, is_secret: l.is_secret, reveal: false, state: "new" },
      );
    } else if (b) {
      rows.push({ key, value: b.value, is_secret: b.is_secret, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "inherited" });
    }
  }
  rows.sort((a, b) => a.key.localeCompare(b.key));
  return rows;
}

function secretMap(rows: Row[]): Map<string, boolean> {
  return new Map(rows.map((r) => [r.key, r.is_secret]));
}

function parseRawLines(text: string): { key: string; value: string }[] {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"))
    .map((l) => {
      const i = l.indexOf("=");
      return { key: (i < 0 ? l : l.slice(0, i)).trim(), value: i < 0 ? "" : l.slice(i + 1) };
    });
}

// classify turns raw KEY=value lines back into rows, re-deriving each one's origin
// against base: a line equal to base inherits, a differing line overrides, an
// unknown line is new, and a base key absent from the text becomes a tombstone.
function classify(
  parsed: { key: string; value: string }[],
  baseMap: Map<string, BaseVal>,
  prevSecret: Map<string, boolean>,
  isBase: boolean,
  defaultSecret: boolean,
): Row[] {
  if (isBase) {
    return parsed.map((p) => ({
      key: p.key,
      value: p.value,
      is_secret: prevSecret.get(p.key) ?? defaultSecret,
      reveal: false,
      state: "base" as const,
    }));
  }
  const seen = new Set(parsed.map((p) => p.key));
  const rows: Row[] = [];
  for (const p of parsed) {
    const b = baseMap.get(p.key);
    const is_secret = prevSecret.get(p.key) ?? b?.is_secret ?? defaultSecret;
    if (b) {
      rows.push(
        p.value === b.value && is_secret === b.is_secret
          ? { key: p.key, value: b.value, is_secret: b.is_secret, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "inherited" }
          : { key: p.key, value: p.value, is_secret, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "override" },
      );
    } else {
      rows.push({ key: p.key, value: p.value, is_secret, reveal: false, state: "new" });
    }
  }
  for (const [key, b] of baseMap) {
    if (!seen.has(key)) {
      rows.push({ key, value: "", is_secret: false, reveal: false, baseValue: b.value, baseSecret: b.is_secret, state: "tombstone" });
    }
  }
  rows.sort((a, b) => a.key.localeCompare(b.key));
  return rows;
}
