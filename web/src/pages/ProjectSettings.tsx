import { FormEvent, useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import {
  Box,
  Button,
  Flash,
  FormControl,
  Heading,
  Label,
  Select,
  Spinner,
  Text,
  TextInput,
  UnderlineNav,
} from "../ui/primer";
import { api, canWrite, Environment, Project, Token as APIToken } from "../api";
import { useToast } from "../lib/toast";
import EnvironmentManager from "../components/EnvironmentManager";
import MembersManager from "../components/MembersManager";

const TABS = ["members", "environments", "tokens"] as const;
type SettingsTab = (typeof TABS)[number];
const TAB_LABELS: Record<SettingsTab, string> = {
  members: "Members",
  environments: "Environments",
  tokens: "Service tokens",
};

export default function ProjectSettings() {
  const { slug = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const [project, setProject] = useState<Project | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api
      .get<Project>(`/projects/${slug}`)
      .then(setProject)
      .catch((e: any) => setErr(e.message));
  }, [slug]);

  if (!project) return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;

  const raw = params.get("tab");
  const tab: SettingsTab = (TABS as readonly string[]).includes(raw ?? "") ? (raw as SettingsTab) : "members";
  const setTab = (t: SettingsTab) => setParams(t === "members" ? {} : { tab: t }, { replace: true });
  const writable = canWrite(project.your_role);

  const readOnlyNote = (
    <Text sx={{ color: "fg.muted" }}>
      You have read-only (viewer) access — this is limited to editors and owners.
    </Text>
  );

  return (
    <Box sx={{ display: "grid", gap: 3 }}>
      <Heading sx={{ fontSize: 4 }}>{project.name} settings</Heading>

      <UnderlineNav aria-label="Project settings">
        {TABS.map((t) => (
          <UnderlineNav.Item
            key={t}
            aria-current={tab === t ? "page" : undefined}
            onSelect={(e) => {
              e.preventDefault();
              setTab(t);
            }}
          >
            {TAB_LABELS[t]}
          </UnderlineNav.Item>
        ))}
      </UnderlineNav>

      {tab === "members" && <MembersManager slug={slug} role={project.your_role} />}
      {tab === "environments" && (writable ? <EnvironmentManager slug={slug} /> : readOnlyNote)}
      {tab === "tokens" && (writable ? <TokensSection slug={slug} /> : readOnlyNote)}
    </Box>
  );
}

function TokensSection({ slug }: { slug: string }) {
  const toast = useToast();
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
      toast("Service token created.");
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function revoke(id: string) {
    setErr("");
    try {
      await api.del(`/projects/${slug}/tokens/${id}`);
      await load();
      toast("Service token revoked.");
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
            sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted", display: "flex", gap: 2, alignItems: "center" }}
          >
            <Text sx={{ fontWeight: "bold" }}>{t.name}</Text>
            <Text sx={{ fontFamily: "mono", color: "fg.muted" }}>{t.prefix}…</Text>
            <Label>{t.scope}</Label>
            {t.revoked_at ? <Label variant="danger">revoked</Label> : <Label variant="success">active</Label>}
            <Box sx={{ flex: 1 }} />
            {!t.revoked_at && (
              <Button variant="danger" size="small" onClick={() => revoke(t.id)}>
                Revoke
              </Button>
            )}
          </Box>
        ))}
      </Box>
    </Box>
  );
}
