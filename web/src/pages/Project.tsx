import { ChangeEvent, FormEvent, useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import {
  Box,
  Button,
  FormControl,
  Flash,
  Heading,
  Label,
  Select,
  Spinner,
  Text,
  TextInput,
  Textarea,
  Token,
} from "@primer/react";
import { api, Config, DiffResult, Environment, Item, Project, Token as APIToken, VersionMeta } from "../api";

const fileFormats = ["json", "csv", "xml"];
const fileContentType: Record<string, string> = {
  json: "application/json",
  csv: "text/csv",
  xml: "application/xml",
};

function parseItems(text: string): Item[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"))
    .map((l) => {
      const i = l.indexOf("=");
      return {
        key: (i < 0 ? l : l.slice(0, i)).trim(),
        value: i < 0 ? "" : l.slice(i + 1),
        is_secret: false,
        deleted: false,
      };
    });
}

const formatItems = (items: Item[]) =>
  items
    .filter((i) => !i.deleted)
    .map((i) => `${i.key}=${i.value}`)
    .join("\n");

export default function ProjectPage() {
  const { slug = "" } = useParams();
  const [project, setProject] = useState<Project | null>(null);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [err, setErr] = useState("");

  async function loadAll() {
    try {
      const [p, e, c, t] = await Promise.all([
        api.get<Project>(`/projects/${slug}`),
        api.get<Environment[]>(`/projects/${slug}/environments`),
        api.get<Config[]>(`/projects/${slug}/configs`),
        api.get<APIToken[]>(`/projects/${slug}/tokens`),
      ]);
      setProject(p);
      setEnvs(e);
      setConfigs(c);
      setTokens(t);
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void loadAll();
  }, [slug]);

  if (!project) {
    return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;
  }

  return (
    <Box sx={{ display: "grid", gap: 4 }}>
      <Box>
        <Heading sx={{ fontSize: 4 }}>{project.name}</Heading>
        <Box sx={{ mt: 2 }}>
          {envs.map((e) => (
            <Label key={e.id} sx={{ mr: 2 }} variant={e.is_default ? "accent" : "secondary"}>
              {e.slug}
            </Label>
          ))}
        </Box>
      </Box>
      {err && <Flash variant="danger">{err}</Flash>}

      <ConfigsSection slug={slug} configs={configs} envs={envs} onChange={loadAll} />
      <ResolvedSection slug={slug} envs={envs} />
      <TokensSection slug={slug} tokens={tokens} envs={envs} onChange={loadAll} />
    </Box>
  );
}

function ConfigsSection({
  slug,
  configs,
  envs,
  onChange,
}: {
  slug: string;
  configs: Config[];
  envs: Environment[];
  onChange: () => void;
}) {
  const [name, setName] = useState("");
  const [format, setFormat] = useState("env");
  const [selected, setSelected] = useState<Config | null>(null);
  const [layer, setLayer] = useState("base");
  const [text, setText] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  const layers = ["base", ...envs.map((e) => e.slug)];
  const isFile = (c: Config | null) => !!c && c.kind === "file";

  async function createConfig(e: FormEvent) {
    e.preventDefault();
    setErr("");
    const kind = fileFormats.includes(format) ? "file" : "variable";
    try {
      await api.post(`/projects/${slug}/configs`, { kind, format, name });
      setName("");
      onChange();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function open(c: Config, lyr: string) {
    setSelected(c);
    setLayer(lyr);
    setMsg("");
    setErr("");
    setShowHistory(false);
    try {
      if (c.kind === "file") {
        setText(await api.getText(`/projects/${slug}/configs/${c.id}/blob?env=${lyr}`).catch(() => ""));
      } else {
        const items = await api.get<Item[]>(`/projects/${slug}/configs/${c.id}/items?env=${lyr}`);
        setText(formatItems(items));
      }
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function save() {
    if (!selected) return;
    setErr("");
    setMsg("");
    try {
      if (selected.kind === "file") {
        await api.putRaw(
          `/projects/${slug}/configs/${selected.id}/blob?env=${layer}`,
          text,
          fileContentType[selected.format] || "application/octet-stream",
        );
      } else {
        await api.put(`/projects/${slug}/configs/${selected.id}/items?env=${layer}`, { items: parseItems(text) });
      }
      setMsg(`Saved ${layer}.`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  function importEnv(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (!f) return;
    f.text().then((content) => setText((prev) => (prev ? prev + "\n" : "") + content));
    e.target.value = "";
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Configs</Heading>
      <Box as="form" onSubmit={createConfig} sx={{ display: "flex", gap: 2, mb: 3, alignItems: "flex-end", flexWrap: "wrap" }}>
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="app or seed.json" />
        </FormControl>
        <FormControl>
          <FormControl.Label>Type</FormControl.Label>
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            <Select.Option value="env">env (variables)</Select.Option>
            <Select.Option value="properties">properties (variables)</Select.Option>
            <Select.Option value="secret">secret (variables)</Select.Option>
            <Select.Option value="json">json (file)</Select.Option>
            <Select.Option value="csv">csv (file)</Select.Option>
            <Select.Option value="xml">xml (file)</Select.Option>
          </Select>
        </FormControl>
        <Button type="submit">Add config</Button>
      </Box>

      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {configs.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No configs yet.</Box>}
        {configs.map((c) => (
          <Box key={c.id} sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted" }}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 2, flexWrap: "wrap" }}>
              <Text sx={{ fontWeight: "bold" }}>{c.name}</Text>
              <Label variant="secondary">{c.format}</Label>
              <Label variant={c.kind === "file" ? "accent" : "default"}>{c.kind}</Label>
              {c.tags.map((t) => (
                <Token key={t} text={t} />
              ))}
              <Box sx={{ flex: 1 }} />
              {layers.map((l) => (
                <Button
                  key={l}
                  size="small"
                  variant={selected?.id === c.id && layer === l ? "primary" : "default"}
                  onClick={() => open(c, l)}
                >
                  {l}
                </Button>
              ))}
            </Box>
            {selected?.id === c.id && (
              <Box sx={{ mt: 3 }}>
                <Box sx={{ display: "flex", alignItems: "center", mb: 1 }}>
                  <Text sx={{ color: "fg.muted" }}>
                    Editing <b>{layer}</b> layer{isFile(selected) ? " (file content)" : " (KEY=VALUE per line)"}
                  </Text>
                  <Box sx={{ flex: 1 }} />
                  <Button size="small" variant="invisible" onClick={() => setShowHistory((v) => !v)}>
                    {showHistory ? "Hide history" : "History"}
                  </Button>
                </Box>
                <Textarea
                  rows={isFile(selected) ? 10 : 6}
                  block
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  sx={{ fontFamily: "mono" }}
                />
                <Box sx={{ mt: 2, display: "flex", gap: 2, alignItems: "center" }}>
                  <Button variant="primary" onClick={save}>
                    Save {layer}
                  </Button>
                  {!isFile(selected) && (
                    <>
                      <Button size="small" onClick={() => fileInput.current?.click()}>
                        Import .env
                      </Button>
                      <input ref={fileInput} type="file" accept=".env,text/plain" hidden onChange={importEnv} />
                    </>
                  )}
                  {msg && <Text sx={{ color: "success.fg" }}>{msg}</Text>}
                  {err && <Text sx={{ color: "danger.fg" }}>{err}</Text>}
                </Box>
                {showHistory && (
                  <VersionHistory slug={slug} config={selected} layer={layer} onRolledBack={() => open(selected, layer)} />
                )}
              </Box>
            )}
          </Box>
        ))}
      </Box>
    </Box>
  );
}

function VersionHistory({
  slug,
  config,
  layer,
  onRolledBack,
}: {
  slug: string;
  config: Config;
  layer: string;
  onRolledBack: () => void;
}) {
  const [versions, setVersions] = useState<VersionMeta[]>([]);
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [err, setErr] = useState("");

  async function load() {
    try {
      setVersions(await api.get<VersionMeta[]>(`/projects/${slug}/configs/${config.id}/versions?env=${layer}`));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
  }, [config.id, layer]);

  async function showDiff(v: number) {
    setErr("");
    try {
      setDiff(await api.get<DiffResult>(`/projects/${slug}/configs/${config.id}/diff?env=${layer}&from=${v}&to=0`));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function rollback(v: number) {
    setErr("");
    try {
      await api.post(`/projects/${slug}/configs/${config.id}/rollback`, { env: layer, to_version: v });
      await load();
      onRolledBack();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box sx={{ mt: 3, p: 3, bg: "canvas.subtle", borderRadius: 2 }}>
      <Text sx={{ fontWeight: "bold", display: "block", mb: 2 }}>History</Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {versions.length === 0 && <Text sx={{ color: "fg.muted" }}>No versions yet.</Text>}
      {versions.map((v) => (
        <Box key={v.version} sx={{ display: "flex", alignItems: "center", gap: 2, py: 1 }}>
          <Text sx={{ fontFamily: "mono" }}>v{v.version}</Text>
          {v.is_current && <Label variant="success">current</Label>}
          <Text sx={{ color: "fg.muted", fontSize: 0 }}>{new Date(v.created_at).toLocaleString()}</Text>
          {v.comment && <Text sx={{ color: "fg.muted", fontSize: 0 }}>· {v.comment}</Text>}
          <Box sx={{ flex: 1 }} />
          <Button size="small" variant="invisible" onClick={() => showDiff(v.version)}>
            Diff vs current
          </Button>
          {!v.is_current && (
            <Button size="small" onClick={() => rollback(v.version)}>
              Rollback
            </Button>
          )}
        </Box>
      ))}
      {diff && (
        <Box sx={{ mt: 2 }}>
          <Text sx={{ fontWeight: "bold", display: "block", mb: 1 }}>
            Diff v{diff.from} → v{diff.to}
          </Text>
          {diff.kind === "variable" ? (
            (diff.vars || []).length === 0 ? (
              <Text sx={{ color: "fg.muted" }}>No differences.</Text>
            ) : (
              (diff.vars || []).map((d) => (
                <Box key={d.key} sx={{ fontFamily: "mono", fontSize: 0 }}>
                  <Label
                    sx={{ mr: 2 }}
                    variant={d.status === "added" ? "success" : d.status === "removed" ? "danger" : "attention"}
                  >
                    {d.status}
                  </Label>
                  {d.key}: {d.from ?? "∅"} → {d.to ?? "∅"}
                </Box>
              ))
            )
          ) : (
            <Box as="pre" sx={{ fontFamily: "mono", fontSize: 0, whiteSpace: "pre-wrap", overflow: "auto" }}>
              {diff.file_diff || "(no changes)"}
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
}

function ResolvedSection({ slug, envs }: { slug: string; envs: Environment[] }) {
  const [env, setEnv] = useState(envs[0]?.slug ?? "development");
  const [out, setOut] = useState("");

  async function load() {
    setOut(await api.resolveText(slug, env, "dotenv"));
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Resolved</Heading>
      <Box sx={{ display: "flex", gap: 2, alignItems: "flex-end", mb: 2 }}>
        <FormControl>
          <FormControl.Label>Environment</FormControl.Label>
          <Select value={env} onChange={(e) => setEnv(e.target.value)}>
            {envs.map((e) => (
              <Select.Option key={e.id} value={e.slug}>
                {e.slug}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <Button onClick={load}>Resolve</Button>
      </Box>
      {out && (
        <Box
          as="pre"
          sx={{ p: 3, bg: "canvas.subtle", borderRadius: 2, fontFamily: "mono", fontSize: 1, overflow: "auto", whiteSpace: "pre-wrap" }}
        >
          {out}
        </Box>
      )}
    </Box>
  );
}

function TokensSection({
  slug,
  tokens,
  envs,
  onChange,
}: {
  slug: string;
  tokens: APIToken[];
  envs: Environment[];
  onChange: () => void;
}) {
  const [name, setName] = useState("");
  const [env, setEnv] = useState(envs[0]?.slug ?? "staging");
  const [minted, setMinted] = useState("");
  const [err, setErr] = useState("");

  async function mint(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setMinted("");
    try {
      const res = await api.post<{ token: string }>(`/projects/${slug}/tokens`, {
        name,
        scope: "read",
        environments: [env],
      });
      setMinted(res.token);
      setName("");
      onChange();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Service tokens</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 2 }}>
        Read-only, project+environment scoped — for CI <code>opentdm pull</code>.
      </Text>
      {minted && (
        <Flash variant="warning" sx={{ mb: 3 }}>
          Copy your token now — it won't be shown again:
          <Box as="code" sx={{ display: "block", mt: 1, fontFamily: "mono", wordBreak: "break-all" }}>
            {minted}
          </Box>
        </Flash>
      )}
      <Box as="form" onSubmit={mint} sx={{ display: "flex", gap: 2, alignItems: "flex-end", mb: 3, flexWrap: "wrap" }}>
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="ci-staging" />
        </FormControl>
        <FormControl>
          <FormControl.Label>Environment</FormControl.Label>
          <Select value={env} onChange={(e) => setEnv(e.target.value)}>
            {envs.map((e) => (
              <Select.Option key={e.id} value={e.slug}>
                {e.slug}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <Button type="submit" variant="primary">
          Create token
        </Button>
      </Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {tokens.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No tokens.</Box>}
        {tokens.map((t) => (
          <Box
            key={t.id}
            sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted", display: "flex", gap: 2 }}
          >
            <Text sx={{ fontWeight: "bold" }}>{t.name}</Text>
            <Text sx={{ fontFamily: "mono", color: "fg.muted" }}>{t.prefix}…</Text>
            <Label>{t.scope}</Label>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
