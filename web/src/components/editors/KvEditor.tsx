import { ChangeEvent, useEffect, useRef, useState } from "react";
import { Box, Button, Flash, IconButton, Spinner, Text, TextInput, Textarea } from "@primer/react";
import { EyeClosedIcon, EyeIcon, LockIcon, TrashIcon } from "@primer/octicons-react";
import { api, Config, Item } from "../../api";

const keyRe = /^[A-Za-z_][A-Za-z0-9_]*$/;

interface Row {
  key: string;
  value: string;
  is_secret: boolean;
  reveal: boolean;
}

interface KvEditorProps {
  slug: string;
  config: Config;
  layer: string;
}

export default function KvEditor({ slug, config, layer }: KvEditorProps) {
  const [rows, setRows] = useState<Row[]>([]);
  const [raw, setRaw] = useState(false);
  const [rawText, setRawText] = useState("");
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMsg("");
    setErr("");
    setRaw(false);
    setLoading(true);
    api
      .getItems(slug, config.id, layer)
      .then((items: Item[]) =>
        setRows(items.filter((i) => !i.deleted).map((i) => ({ key: i.key, value: i.value, is_secret: i.is_secret, reveal: false }))),
      )
      .catch((e: any) => setErr(e.message))
      .finally(() => setLoading(false));
  }, [config.id, layer]);

  function update(i: number, patch: Partial<Row>) {
    setRows((rs) => rs.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }
  function addRow() {
    setRows((rs) => [...rs, { key: "", value: "", is_secret: config.format === "secret", reveal: true }]);
  }

  function toRaw() {
    setRawText(rows.map((r) => `${r.key}=${r.value}`).join("\n"));
    setRaw(true);
  }
  function fromRaw() {
    setRows(parseDotenv(rawText, secretMap(rows), config.format === "secret"));
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
    const data = raw ? parseDotenv(rawText, secretMap(rows), config.format === "secret") : rows;
    for (const r of data) {
      if (!keyRe.test(r.key)) {
        setErr(`Invalid key: ${r.key || "(empty)"} — must match [A-Za-z_][A-Za-z0-9_]*`);
        return;
      }
    }
    try {
      // Send is_secret per row — this is the fix vs the old flatten-to-text path,
      // which silently de-secreted every value on save.
      await api.putItems(
        slug,
        config.id,
        layer,
        data.map((r) => ({ key: r.key, value: r.value, is_secret: r.is_secret, deleted: false })),
      );
      setRows(data.map((r) => ({ ...r, reveal: false })));
      setRaw(false);
      setMsg(`Saved ${layer}.`);
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
        <Textarea
          block
          rows={10}
          value={rawText}
          onChange={(e) => setRawText(e.target.value)}
          sx={{ fontFamily: "mono" }}
        />
      ) : (
        <Box sx={{ display: "grid", gap: 1 }}>
          {rows.length === 0 && <Text sx={{ color: "fg.muted" }}>No variables in this layer.</Text>}
          {rows.map((r, i) => (
            <Box key={i} sx={{ display: "flex", gap: 1, alignItems: "center" }}>
              <TextInput
                value={r.key}
                onChange={(e) => update(i, { key: e.target.value })}
                placeholder="KEY"
                sx={{ width: 240, fontFamily: "mono" }}
              />
              <TextInput
                block
                value={r.value}
                onChange={(e) => update(i, { value: e.target.value })}
                placeholder="value"
                type={r.is_secret && !r.reveal ? "password" : "text"}
                sx={{ flex: 1, fontFamily: "mono" }}
              />
              <IconButton
                icon={LockIcon}
                aria-label="mark secret"
                size="small"
                variant={r.is_secret ? "primary" : "default"}
                onClick={() => update(i, { is_secret: !r.is_secret })}
              />
              {r.is_secret && (
                <IconButton
                  icon={r.reveal ? EyeIcon : EyeClosedIcon}
                  aria-label="reveal value"
                  size="small"
                  onClick={() => update(i, { reveal: !r.reveal })}
                />
              )}
              <IconButton
                icon={TrashIcon}
                aria-label="delete"
                size="small"
                variant="danger"
                onClick={() => setRows((rs) => rs.filter((_, idx) => idx !== i))}
              />
            </Box>
          ))}
        </Box>
      )}
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
      <Text sx={{ color: "fg.muted", fontSize: 0, display: "block", mt: 2 }}>
        Editing the <b>{layer}</b> layer. Keys not set here inherit from base — see the project's Resolved view.
      </Text>
    </Box>
  );
}

function secretMap(rows: Row[]): Map<string, boolean> {
  return new Map(rows.map((r) => [r.key, r.is_secret]));
}

function parseDotenv(text: string, prevSecret: Map<string, boolean>, defaultSecret: boolean): Row[] {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"))
    .map((l) => {
      const i = l.indexOf("=");
      const key = (i < 0 ? l : l.slice(0, i)).trim();
      return { key, value: i < 0 ? "" : l.slice(i + 1), is_secret: prevSecret.get(key) ?? defaultSecret, reveal: false };
    });
}
