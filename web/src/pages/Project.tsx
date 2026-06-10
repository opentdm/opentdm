import { useEffect, useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { Box, Button, Dialog, Flash, Heading, IconButton, Label, Spinner, Text } from "../ui/primer";
import {
  ChevronRightIcon,
  EyeIcon,
  FileIcon,
  GearIcon,
  KeyIcon,
  PlusIcon,
  PulseIcon,
  StackIcon,
} from "@primer/octicons-react";
import { api, canWrite, Config, Environment, Project } from "../api";
import { errMessage } from "../lib/errors";
import ResolvedView from "../components/ResolvedView";
import AddObjectDialog from "../components/AddObjectDialog";
import Overline from "../components/Overline";

export default function ProjectPage() {
  const { slug = "" } = useParams();
  const [project, setProject] = useState<Project | null>(null);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [memberCount, setMemberCount] = useState<number | null>(null);
  const [err, setErr] = useState("");

  async function loadAll() {
    try {
      const [p, e, c, m] = await Promise.all([
        api.get<Project>(`/projects/${slug}`),
        api.listEnvs(slug),
        api.get<Config[]>(`/projects/${slug}/configs`),
        api.listMembers(slug),
      ]);
      setProject(p);
      setEnvs(e);
      setConfigs(c);
      setMemberCount(m.length);
    } catch (e) {
      setErr(errMessage(e));
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
          <Overline>Project</Overline>
          <Heading sx={{ fontSize: 4 }}>{project.name}</Heading>
          {project.description && (
            <Text sx={{ color: "fg.muted", display: "block", mt: 1 }}>{project.description}</Text>
          )}
          <Box sx={{ mt: 2, display: "flex", alignItems: "center", gap: 2, flexWrap: "wrap" }}>
            <Label variant="accent" className="otdm-pill-accent">
              base
            </Label>
            {envs.map((e) => (
              <Label key={e.id} variant="secondary">
                {e.slug}
                {e.is_default ? " · default" : ""}
              </Label>
            ))}
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

      <Box className="otdm-meta-grid">
        <div className="otdm-meta-cell">
          <div className="k">Objects</div>
          <div className="v">{configs.length}</div>
        </div>
        <div className="otdm-meta-cell">
          <div className="k">Environments</div>
          <div className="v">{envs.length}</div>
        </div>
        <div className="otdm-meta-cell">
          <div className="k">Members</div>
          <div className="v">{memberCount ?? "—"}</div>
        </div>
        <div className="otdm-meta-cell">
          <div className="k">Your role</div>
          <div className="v" style={{ textTransform: "capitalize", fontSize: 15, fontWeight: 600 }}>
            {project.your_role ?? "—"}
          </div>
        </div>
      </Box>

      <ObjectsSection
        slug={slug}
        configs={configs}
        envs={envs}
        canWrite={canWrite(project.your_role)}
        onChange={loadAll}
      />
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
  const [adding, setAdding] = useState(false);
  const [resolveTarget, setResolveTarget] = useState<Config | null>(null);

  return (
    <Box>
      <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 2 }}>
        <Box sx={{ color: "fg.muted", display: "flex" }}>
          <StackIcon />
        </Box>
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
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                <Text sx={{ fontWeight: "bold" }}>{c.name}</Text>
                {c.description && <Text sx={{ color: "fg.muted", fontSize: 0 }}>{c.description}</Text>}
              </Box>
              <Label variant="secondary">{c.format}</Label>
            </Box>
            {c.kind === "variable" && <Text sx={{ color: "fg.muted", fontSize: 0 }}>{c.key_count ?? 0} keys</Text>}
            {c.kind === "variable" && (
              <IconButton
                icon={EyeIcon}
                aria-label={`View resolved ${c.name}`}
                variant="invisible"
                onClick={() => setResolveTarget(c)}
              />
            )}
            <Box sx={{ pr: 3, color: "fg.muted", display: "flex" }}>
              <ChevronRightIcon />
            </Box>
          </Box>
        ))}
      </Box>

      {adding && <AddObjectDialog slug={slug} onClose={() => setAdding(false)} onChange={onChange} />}

      {resolveTarget && (
        <Dialog title={`Resolved — ${resolveTarget.name}`} width="large" onClose={() => setResolveTarget(null)}>
          <ResolvedView slug={slug} config={resolveTarget} envs={envs} />
        </Dialog>
      )}
    </Box>
  );
}
