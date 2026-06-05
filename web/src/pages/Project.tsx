import { FormEvent, useEffect, useState } from "react";
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
import { api, Config, Environment, Item, Project, Token as APIToken } from "../api";

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
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  const layers = ["base", ...envs.map((e) => e.slug)];

  async function createConfig(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      await api.post(`/projects/${slug}/configs`, { kind: "variable", format, name });
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
    try {
      const items = await api.get<Item[]>(`/projects/${slug}/configs/${c.id}/items?env=${lyr}`);
      setText(formatItems(items));
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function save() {
    if (!selected) return;
    setErr("");
    setMsg("");
    try {
      await api.put(`/projects/${slug}/configs/${selected.id}/items?env=${layer}`, {
        items: parseItems(text),
      });
      setMsg(`Saved ${layer}.`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Configs</Heading>
      <Box
        as="form"
        onSubmit={createConfig}
        sx={{ display: "flex", gap: 2, mb: 3, alignItems: "flex-end", flexWrap: "wrap" }}
      >
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="app" />
        </FormControl>
        <FormControl>
          <FormControl.Label>Format</FormControl.Label>
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            <Select.Option value="env">env</Select.Option>
            <Select.Option value="properties">properties</Select.Option>
            <Select.Option value="secret">secret</Select.Option>
          </Select>
        </FormControl>
        <Button type="submit">Add config</Button>
      </Box>

      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {configs.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No configs yet.</Box>}
        {configs.map((c) => (
          <Box
            key={c.id}
            sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted" }}
          >
            <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
              <Text sx={{ fontWeight: "bold" }}>{c.name}</Text>
              <Label variant="secondary">{c.format}</Label>
              {c.tags.map((t) => (
                <Token key={t} text={t} />
              ))}
              <Box sx={{ flex: 1 }} />
              {layers.map((l) => (
                <Button key={l} size="small" variant={selected?.id === c.id && layer === l ? "primary" : "default"} onClick={() => open(c, l)}>
                  {l}
                </Button>
              ))}
            </Box>
            {selected?.id === c.id && (
              <Box sx={{ mt: 3 }}>
                <Text sx={{ color: "fg.muted", display: "block", mb: 1 }}>
                  Editing <b>{layer}</b> layer (KEY=VALUE per line)
                </Text>
                <Textarea
                  rows={6}
                  block
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  sx={{ fontFamily: "mono" }}
                />
                <Box sx={{ mt: 2, display: "flex", gap: 2, alignItems: "center" }}>
                  <Button variant="primary" onClick={save}>
                    Save {layer}
                  </Button>
                  {msg && <Text sx={{ color: "success.fg" }}>{msg}</Text>}
                  {err && <Text sx={{ color: "danger.fg" }}>{err}</Text>}
                </Box>
              </Box>
            )}
          </Box>
        ))}
      </Box>
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
          sx={{
            p: 3,
            bg: "canvas.subtle",
            borderRadius: 2,
            fontFamily: "mono",
            fontSize: 1,
            overflow: "auto",
            whiteSpace: "pre-wrap",
          }}
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
