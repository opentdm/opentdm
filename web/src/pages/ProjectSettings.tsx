import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import {
  Box,
  Breadcrumbs,
  Button,
  Flash,
  FormControl,
  Heading,
  Label,
  Select,
  Spinner,
  Text,
  TextInput,
} from "@primer/react";
import { api, Environment, Project, Token as APIToken } from "../api";
import EnvironmentManager from "../components/EnvironmentManager";
import EnvironmentCloner from "../components/EnvironmentCloner";

export default function ProjectSettings() {
  const { slug = "" } = useParams();
  const [project, setProject] = useState<Project | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api
      .get<Project>(`/projects/${slug}`)
      .then(setProject)
      .catch((e: any) => setErr(e.message));
  }, [slug]);

  if (!project) return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;

  return (
    <Box sx={{ display: "grid", gap: 4 }}>
      <Box>
        <Breadcrumbs>
          <Breadcrumbs.Item as={RouterLink} to={`/projects/${slug}`}>
            {slug}
          </Breadcrumbs.Item>
          <Breadcrumbs.Item selected>Settings</Breadcrumbs.Item>
        </Breadcrumbs>
        <Heading sx={{ fontSize: 4, mt: 2 }}>{project.name} settings</Heading>
      </Box>
      <EnvironmentManager slug={slug} />
      <EnvironmentCloner slug={slug} />
      <TokensSection slug={slug} />
    </Box>
  );
}

function TokensSection({ slug }: { slug: string }) {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [name, setName] = useState("");
  const [env, setEnv] = useState("");
  const [minted, setMinted] = useState("");
  const [err, setErr] = useState("");

  async function load() {
    try {
      const [t, e] = await Promise.all([api.get<APIToken[]>(`/projects/${slug}/tokens`), api.listEnvs(slug)]);
      setTokens(t);
      setEnvs(e);
      setEnv((cur) => cur || e[0]?.slug || "");
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
  }, [slug]);

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
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 2 }}>Service tokens</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 2 }}>
        Read-only, project + environment scoped — for CI <code>opentdm pull</code>.
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
