import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink, useNavigate, useParams } from "react-router-dom";
import {
  ActionList,
  ActionMenu,
  Box,
  Button,
  Dialog,
  Flash,
  FormControl,
  Heading,
  IconButton,
  Label,
  Spinner,
  Text,
  TextInput,
} from "../ui/primer";
import { EyeIcon, FileIcon, GearIcon, KebabHorizontalIcon, KeyIcon, PlusIcon, PulseIcon } from "@primer/octicons-react";
import { api, canWrite, Config, Environment, Project } from "../api";
import ResolvedView from "../components/ResolvedView";

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
            <Label variant="primary">base</Label>
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
          <Text sx={{ color: "fg.muted", fontSize: 0, display: "block", mt: 1 }}>
            Shared defaults (<b>base</b>) + {envs.length} environment{envs.length === 1 ? "" : "s"} that override it.
          </Text>
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

      <ObjectsSection slug={slug} configs={configs} envs={envs} canWrite={canWrite(project.your_role)} onChange={loadAll} />
    </Box>
  );
}

function ObjectsSection({
  slug,
  configs,
  envs,
  canWrite,
  onChange,
}: {
  slug: string;
  configs: Config[];
  envs: Environment[];
  canWrite: boolean;
  onChange: () => void;
}) {
  const nav = useNavigate();
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [resolveTarget, setResolveTarget] = useState<Config | null>(null);

  async function createConfig(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      // Env-only: every new object is a variable/env bundle.
      const created = await api.post<Config>(`/projects/${slug}/configs`, {
        kind: "variable",
        format: "env",
        name,
      });
      setName("");
      setAdding(false);
      onChange();
      if (created?.id) nav(`/projects/${slug}/configs/${created.id}`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
        <Heading sx={{ fontSize: 3 }}>Objects</Heading>
        <Box sx={{ flex: 1 }} />
        {canWrite && (
          <Button variant="primary" leadingVisual={PlusIcon} onClick={() => setAdding(true)}>
            Add object
          </Button>
        )}
      </Box>

      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {configs.length === 0 && (
          <Box sx={{ p: 3, color: "fg.muted" }}>No objects yet{canWrite ? " — click Add object." : "."}</Box>
        )}
        {configs.map((c, i) => (
          <Box
            key={c.id}
            className="otdm-hover-row"
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 1,
              borderBottomWidth: i < configs.length - 1 ? 1 : 0,
              borderBottomStyle: "solid",
              borderColor: "border.muted",
            }}
          >
            <Box
              as={RouterLink}
              to={`/projects/${slug}/configs/${c.id}`}
              sx={{
                flex: 1,
                p: 3,
                display: "flex",
                alignItems: "center",
                gap: 2,
                flexWrap: "wrap",
                textDecoration: "none",
                color: "fg.default",
              }}
            >
              <Box sx={{ color: "fg.muted", display: "flex" }}>{c.kind === "file" ? <FileIcon /> : <KeyIcon />}</Box>
              <Text sx={{ fontWeight: "bold" }}>{c.name}</Text>
              <Label variant="secondary">{c.format}</Label>
            </Box>
            {c.kind === "variable" && (
              <Box sx={{ pr: 2 }}>
                <ActionMenu>
                  <ActionMenu.Anchor>
                    <IconButton icon={KebabHorizontalIcon} aria-label={`Actions for ${c.name}`} variant="invisible" />
                  </ActionMenu.Anchor>
                  <ActionMenu.Overlay width="small">
                    <ActionList>
                      <ActionList.Item onSelect={() => setResolveTarget(c)}>
                        <ActionList.LeadingVisual>
                          <EyeIcon />
                        </ActionList.LeadingVisual>
                        View resolved
                      </ActionList.Item>
                    </ActionList>
                  </ActionMenu.Overlay>
                </ActionMenu>
              </Box>
            )}
          </Box>
        ))}
      </Box>

      {adding && (
        <Dialog title="Add object" onClose={() => setAdding(false)}>
          <Box as="form" onSubmit={createConfig} sx={{ display: "grid", gap: 3 }}>
            {err && <Flash variant="danger">{err}</Flash>}
            <FormControl>
              <FormControl.Label>Name</FormControl.Label>
              <TextInput block value={name} onChange={(e) => setName(e.target.value)} autoFocus />
              <FormControl.Caption>An env (.env-style) variable bundle.</FormControl.Caption>
            </FormControl>
            <Box sx={{ display: "flex", gap: 2, justifyContent: "flex-end" }}>
              <Button type="button" onClick={() => setAdding(false)}>
                Cancel
              </Button>
              <Button type="submit" variant="primary">
                Add object
              </Button>
            </Box>
          </Box>
        </Dialog>
      )}

      {resolveTarget && (
        <Dialog title={`Resolved — ${resolveTarget.name}`} width="large" onClose={() => setResolveTarget(null)}>
          <ResolvedView slug={slug} config={resolveTarget} envs={envs} />
        </Dialog>
      )}
    </Box>
  );
}
