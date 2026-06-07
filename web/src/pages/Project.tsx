import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink, useNavigate, useParams } from "react-router-dom";
import {
  Box,
  Button,
  Checkbox,
  Flash,
  FormControl,
  Heading,
  Label,
  Select,
  Spinner,
  Text,
  TextInput,
  Token,
} from "../ui/primer";
import { FileIcon, GearIcon, KeyIcon, PulseIcon } from "@primer/octicons-react";
import { api, canWrite, Collision, Config, Environment, Project } from "../api";

const fileFormats = ["json", "csv", "xml"];

export default function ProjectPage() {
  const { slug = "" } = useParams();
  const [project, setProject] = useState<Project | null>(null);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [err, setErr] = useState("");

  async function loadAll() {
    try {
      const [p, e, c] = await Promise.all([
        api.get<Project>(`/projects/${slug}`),
        api.listEnvs(slug),
        api.get<Config[]>(`/projects/${slug}/configs`),
      ]);
      setProject(p);
      setEnvs(e);
      setConfigs(c);
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
      <Box sx={{ display: "flex", alignItems: "flex-start", flexWrap: "wrap", gap: 2 }}>
        <Box>
          <Heading sx={{ fontSize: 4 }}>{project.name}</Heading>
          {project.description && <Text sx={{ color: "fg.muted", display: "block", mt: 1 }}>{project.description}</Text>}
          <Box sx={{ mt: 2, display: "flex", alignItems: "center", gap: 2, flexWrap: "wrap" }}>
            {envs.map((e) => (
              <Label key={e.id} variant={e.is_default ? "accent" : "secondary"}>
                {e.slug}
              </Label>
            ))}
            {project.your_role && (
              <Label variant="secondary" sx={{ ml: 1 }}>
                you: {project.your_role}
              </Label>
            )}
          </Box>
        </Box>
        <Box sx={{ flex: 1 }} />
        <Button as={RouterLink} to={`/projects/${slug}/activity`} leadingVisual={PulseIcon}>
          Activity
        </Button>
        <Button as={RouterLink} to={`/projects/${slug}/settings`} leadingVisual={GearIcon}>
          Settings
        </Button>
      </Box>
      {err && <Flash variant="danger">{err}</Flash>}

      <ObjectsSection slug={slug} configs={configs} canWrite={canWrite(project.your_role)} onChange={loadAll} />
      <ResolvedSection slug={slug} envs={envs} />
    </Box>
  );
}

function ObjectsSection({
  slug,
  configs,
  canWrite,
  onChange,
}: {
  slug: string;
  configs: Config[];
  canWrite: boolean;
  onChange: () => void;
}) {
  const nav = useNavigate();
  const [name, setName] = useState("");
  const [format, setFormat] = useState("env");
  const [tags, setTags] = useState("");
  const [err, setErr] = useState("");

  async function createConfig(e: FormEvent) {
    e.preventDefault();
    setErr("");
    const kind = fileFormats.includes(format) ? "file" : "variable";
    try {
      const created = await api.post<Config>(`/projects/${slug}/configs`, {
        kind,
        format,
        name,
        tags: tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
      });
      setName("");
      setTags("");
      onChange();
      if (created?.id) nav(`/projects/${slug}/configs/${created.id}`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Objects</Heading>
      {canWrite && (
      <Box
        as="form"
        onSubmit={createConfig}
        sx={{ display: "flex", gap: 2, mb: 3, alignItems: "flex-end", flexWrap: "wrap" }}
      >
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="payments or seed.json" />
        </FormControl>
        <FormControl>
          <FormControl.Label>Type</FormControl.Label>
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            <Select.OptGroup label="Variables">
              <Select.Option value="env">env</Select.Option>
              <Select.Option value="properties">properties</Select.Option>
              <Select.Option value="secret">secret</Select.Option>
            </Select.OptGroup>
            <Select.OptGroup label="Files">
              <Select.Option value="json">json</Select.Option>
              <Select.Option value="csv">csv</Select.Option>
              <Select.Option value="xml">xml</Select.Option>
            </Select.OptGroup>
          </Select>
        </FormControl>
        <FormControl>
          <FormControl.Label>Tags</FormControl.Label>
          <TextInput value={tags} onChange={(e) => setTags(e.target.value)} placeholder="prod, payments" />
        </FormControl>
        <Button type="submit" variant="primary">
          Add object
        </Button>
      </Box>
      )}
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}

      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {configs.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No objects yet — create one above.</Box>}
        {configs.map((c, i) => (
          <Box
            key={c.id}
            as={RouterLink}
            to={`/projects/${slug}/configs/${c.id}`}
            className="otdm-hover-row"
            sx={{
              p: 3,
              display: "flex",
              alignItems: "center",
              gap: 2,
              flexWrap: "wrap",
              textDecoration: "none",
              color: "fg.default",
              borderBottomWidth: i < configs.length - 1 ? 1 : 0,
              borderBottomStyle: "solid",
              borderColor: "border.muted",
            }}
          >
            <Box sx={{ color: "fg.muted", display: "flex" }}>
              {c.kind === "file" ? <FileIcon /> : <KeyIcon />}
            </Box>
            <Text sx={{ fontWeight: "bold" }}>{c.name}</Text>
            <Label variant="secondary">{c.format}</Label>
            {c.tags.map((t) => (
              <Token key={t} text={t} />
            ))}
          </Box>
        ))}
      </Box>
    </Box>
  );
}

function ResolvedSection({ slug, envs }: { slug: string; envs: Environment[] }) {
  const defaultEnv = envs.find((e) => e.is_default)?.slug ?? envs[0]?.slug ?? "";
  const [env, setEnv] = useState(defaultEnv);
  const [format, setFormat] = useState("dotenv");
  // Secrets are hidden by default so the preview is safe to screen-share.
  const [showSecrets, setShowSecrets] = useState(false);
  const [out, setOut] = useState("");
  const [collisions, setCollisions] = useState<Collision[]>([]);
  const [err, setErr] = useState("");

  // Keep the selector pinned to the default env until the user changes it.
  useEffect(() => {
    setEnv(defaultEnv);
  }, [defaultEnv]);

  async function load() {
    setErr("");
    try {
      const [text, meta] = await Promise.all([
        api.resolveText(slug, env, format, showSecrets),
        api.resolveMeta(slug, env),
      ]);
      setOut(text);
      setCollisions(meta.collisions);
    } catch (e: any) {
      setErr(e.message);
    }
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
        <FormControl>
          <FormControl.Label>Format</FormControl.Label>
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            {["dotenv", "json", "shell", "yaml", "properties"].map((f) => (
              <Select.Option key={f} value={f}>
                {f}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <Checkbox checked={showSecrets} onChange={(e) => setShowSecrets(e.target.checked)} />
          <FormControl.Label>Show secrets</FormControl.Label>
        </FormControl>
        <Button onClick={load}>Resolve</Button>
      </Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {collisions.length > 0 && (
        <Flash variant="warning" sx={{ mb: 2 }}>
          <Box sx={{ fontWeight: "bold", mb: 1 }}>
            {collisions.length} cross-config key collision{collisions.length > 1 ? "s" : ""}
          </Box>
          <Box as="ul" sx={{ pl: 3, m: 0 }}>
            {collisions.map((c) => (
              <Box as="li" key={c.key}>
                <Box as="code">{c.key}</Box> — kept <Box as="code">{c.winning_config}</Box>, shadowed{" "}
                <Box as="code">{c.losing_config}</Box>
              </Box>
            ))}
          </Box>
        </Flash>
      )}
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
