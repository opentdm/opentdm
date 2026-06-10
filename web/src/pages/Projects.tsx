import { FormEvent, Fragment, useMemo, useState } from "react";
import { Box, Button, Dialog, Flash, FormControl, Heading, Select, Spinner, TextInput } from "../ui/primer";
import { PlusIcon, SearchIcon, StarFillIcon } from "@primer/octicons-react";
import { Project } from "../api";
import { api } from "../api";
import { useProjectsCtx } from "../lib/projects";
import { useFavourites } from "../lib/favourites";
import { useToast } from "../lib/toast";
import ProjectCard from "../components/ProjectCard";
import { Overline } from "../components/ui";

type SortKey = "name" | "role" | "recent";
type GroupKey = "none" | "role";

const ROLE_RANK: Record<string, number> = { owner: 0, editor: 1, viewer: 2 };
const ROLE_ORDER = ["owner", "editor", "viewer"];

function sortProjects(list: Project[], sort: SortKey): Project[] {
  const copy = list.slice();
  if (sort === "name") copy.sort((a, b) => a.name.localeCompare(b.name));
  else if (sort === "role")
    copy.sort((a, b) => (ROLE_RANK[a.your_role ?? "viewer"] ?? 9) - (ROLE_RANK[b.your_role ?? "viewer"] ?? 9));
  else if (sort === "recent") copy.sort((a, b) => (b.created_at ?? "").localeCompare(a.created_at ?? ""));
  return copy;
}

export default function Projects() {
  const { projects, loading, error, reload } = useProjectsCtx();
  const { favs, toggle } = useFavourites();
  const [q, setQ] = useState("");
  const [sort, setSort] = useState<SortKey>("name");
  const [groupBy, setGroupBy] = useState<GroupKey>("none");
  const [showNew, setShowNew] = useState(false);

  const filtered = useMemo(() => {
    const ql = q.trim().toLowerCase();
    const matched = ql
      ? projects.filter((p) => `${p.name} ${p.slug} ${p.description}`.toLowerCase().includes(ql))
      : projects.slice();
    return sortProjects(matched, sort);
  }, [projects, q, sort]);

  const pinned = filtered.filter((p) => favs.has(p.slug));
  const rest = filtered.filter((p) => !favs.has(p.slug));

  const card = (p: Project) => <ProjectCard key={p.id} project={p} isFav={favs.has(p.slug)} onToggleFav={toggle} />;

  return (
    <Box>
      <Box className="otdm-page-hd">
        <Box className="grow">
          <Overline>Workspace</Overline>
          <Heading sx={{ fontSize: 5 }}>Projects</Heading>
          <Box className="otdm-sub">Typed config, fixtures &amp; secrets — scoped per project, per environment.</Box>
        </Box>
        <Button variant="primary" leadingVisual={PlusIcon} onClick={() => setShowNew(true)}>
          New project
        </Button>
      </Box>

      {error && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {error}
        </Flash>
      )}

      <Box className="otdm-toolbar">
        <Box className="otdm-toolbar-search">
          <TextInput
            block
            leadingVisual={SearchIcon}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={`Filter ${projects.length} project${projects.length === 1 ? "" : "s"}…`}
            aria-label="Filter projects"
          />
        </Box>
        <Select value={sort} onChange={(e) => setSort(e.target.value as SortKey)} aria-label="Sort projects">
          <Select.Option value="name">Sort: Name</Select.Option>
          <Select.Option value="role">Sort: Your role</Select.Option>
          <Select.Option value="recent">Sort: Recent</Select.Option>
        </Select>
        <Select value={groupBy} onChange={(e) => setGroupBy(e.target.value as GroupKey)} aria-label="Group projects">
          <Select.Option value="none">Group: None</Select.Option>
          <Select.Option value="role">Group: By role</Select.Option>
        </Select>
        <Box className="otdm-toolbar-count">
          {filtered.length} of {projects.length}
        </Box>
      </Box>

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", pt: 5 }}>
          <Spinner />
        </Box>
      ) : filtered.length === 0 ? (
        <Box
          sx={{
            mt: 3,
            p: 5,
            textAlign: "center",
            color: "fg.muted",
            borderWidth: 1,
            borderStyle: "solid",
            borderColor: "border.default",
            borderRadius: 2,
          }}
        >
          {projects.length === 0 ? "No projects yet — create one to get started." : `Nothing matches “${q}”.`}
        </Box>
      ) : (
        <>
          {pinned.length > 0 && (
            <>
              <Box className="otdm-section-hd">
                <span className="otdm-sb-ico">
                  <StarFillIcon size={16} fill="#d4a72c" />
                </span>
                <h2>Pinned</h2>
                <span className="count">{pinned.length}</span>
              </Box>
              <Box className="otdm-pgrid">{pinned.map(card)}</Box>
            </>
          )}

          {groupBy === "role" ? (
            ROLE_ORDER.map((r) => {
              const items = rest.filter((p) => (p.your_role ?? "viewer") === r);
              if (items.length === 0) return null;
              return (
                <Fragment key={r}>
                  <Box className="otdm-section-hd">
                    <h2 style={{ textTransform: "capitalize" }}>{r} access</h2>
                    <span className="count">{items.length}</span>
                  </Box>
                  <Box className="otdm-pgrid">{items.map(card)}</Box>
                </Fragment>
              );
            })
          ) : (
            <>
              {pinned.length > 0 && (
                <Box className="otdm-section-hd">
                  <h2>All projects</h2>
                  <span className="count">{rest.length}</span>
                </Box>
              )}
              <Box className="otdm-pgrid" sx={{ mt: pinned.length > 0 ? 0 : 3 }}>
                {rest.map(card)}
              </Box>
            </>
          )}
        </>
      )}

      {showNew && <NewProjectDialog onClose={() => setShowNew(false)} onCreated={reload} />}
    </Box>
  );
}

function NewProjectDialog({ onClose, onCreated }: { onClose: () => void; onCreated: () => Promise<void> }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/projects", { name });
      await onCreated();
      onClose();
      toast("Project created.");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Failed to create project");
      setBusy(false);
    }
  }

  return (
    <Dialog title="New project" onClose={onClose}>
      <Box as="form" onSubmit={create} sx={{ display: "grid", gap: 3 }}>
        {err && <Flash variant="danger">{err}</Flash>}
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput block value={name} onChange={(e) => setName(e.target.value)} placeholder="Payments" autoFocus />
          <FormControl.Caption>The URL slug is derived from the name.</FormControl.Caption>
        </FormControl>
        <Box sx={{ display: "flex", gap: 2, justifyContent: "flex-end" }}>
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={busy || !name.trim()}>
            {busy ? "Creating…" : "Create project"}
          </Button>
        </Box>
      </Box>
    </Dialog>
  );
}
