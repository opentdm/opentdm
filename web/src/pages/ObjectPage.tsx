import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink, useNavigate, useParams } from "react-router-dom";
import {
  ActionList,
  ActionMenu,
  Box,
  Breadcrumbs,
  Button,
  ConfirmationDialog,
  Flash,
  FormControl,
  Heading,
  Label,
  Spinner,
  Text,
  TextInput,
} from "../ui/primer";
import { ColumnsIcon, CopyIcon, EyeClosedIcon, EyeIcon, KebabHorizontalIcon, PencilIcon, TrashIcon } from "@primer/octicons-react";
import { api, canWrite, Config, Environment, Project } from "../api";
import EditorDispatch from "../components/editors/EditorDispatch";
import VersionHistory from "../components/VersionHistory";
import FileTree from "../components/filebrowser/FileTree";
import BranchEnvMenu from "../components/filebrowser/BranchEnvMenu";
import CodeFileView from "../components/filebrowser/CodeFileView";
import DeltaBadge from "../components/filebrowser/DeltaBadge";
import SplitCompare, { Pane } from "../components/filebrowser/SplitCompare";

// The object page is a GitHub-style file browser: the project's objects are a file
// tree on the left; the right shows the selected object resolved for a chosen
// environment (picked with a branch-style dropdown), read-only by default with an
// Edit toggle into the inherited-aware editor. Version history sits below.
export default function ObjectPage() {
  const { slug = "", configId = "" } = useParams();
  const nav = useNavigate();
  const [project, setProject] = useState<Project | null>(null);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [config, setConfig] = useState<Config | null>(null);
  const [env, setEnv] = useState("base");
  const [editing, setEditing] = useState(false);
  const [reveal, setReveal] = useState(false);
  const [split, setSplit] = useState(false);
  const [panes, setPanes] = useState<Pane[]>([{ env: "base" }, { env: "base" }]);
  const [refresh, setRefresh] = useState(0);
  const [copyText, setCopyText] = useState("");
  const [copied, setCopied] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [err, setErr] = useState("");

  // Project-scoped data (tree, envs, role) reloads only when the project changes.
  useEffect(() => {
    Promise.all([api.getProject(slug), api.get<Config[]>(`/projects/${slug}/configs`), api.listEnvs(slug)])
      .then(([p, cs, es]) => {
        setProject(p);
        setConfigs(cs);
        setEnvs(es);
      })
      .catch((e: any) => setErr(e.message));
  }, [slug]);

  // The selected object reloads when navigating between files in the tree.
  useEffect(() => {
    setEditing(false);
    setReveal(false);
    setSplit(false);
    api
      .getConfig(slug, configId)
      .then(setConfig)
      .catch((e: any) => setErr(e.message));
  }, [slug, configId]);

  async function remove() {
    setConfirmDelete(false);
    try {
      await api.archiveConfig(slug, configId);
      nav(`/projects/${slug}`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(copyText);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable */
    }
  }

  const bump = () => setRefresh((n) => n + 1);

  // Enter split with two panes (the current env + the first other one); exit returns
  // to the single view.
  function toggleSplit() {
    if (split) {
      setSplit(false);
      return;
    }
    setEditing(false);
    const other = ["base", ...envs.map((e) => e.slug)].find((e) => e !== env) || "base";
    setPanes([{ env }, { env: other }]);
    setSplit(true);
  }

  if (!config || !project) {
    return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;
  }

  const readOnly = !canWrite(project.your_role);
  const isVar = config.kind === "variable";

  return (
    <Box sx={{ display: "grid", gap: 3 }}>
      <Breadcrumbs>
        <Breadcrumbs.Item as={RouterLink} to={`/projects/${slug}`}>
          {slug}
        </Breadcrumbs.Item>
        <Breadcrumbs.Item selected>{config.name}</Breadcrumbs.Item>
      </Breadcrumbs>

      {err && <Flash variant="danger">{err}</Flash>}
      {readOnly && <Flash>You have read-only (viewer) access to this project.</Flash>}

      {/* Object header */}
      <Box sx={{ display: "flex", alignItems: "center", gap: 2, flexWrap: "wrap" }}>
        <Heading sx={{ fontSize: 4 }}>{config.name}</Heading>
        <Label variant="secondary">{config.format}</Label>
        <Label variant={config.kind === "file" ? "accent" : "default"}>{config.kind}</Label>
        {config.format === "secret" && <Label variant="danger">secret</Label>}
        <Box sx={{ flex: 1 }} />
        {!readOnly && (
          <ActionMenu>
            <ActionMenu.Anchor>
              <Button leadingVisual={KebabHorizontalIcon}>Object</Button>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay width="small">
              <ActionList>
                <ActionList.Item onSelect={() => setEditingName((v) => !v)}>
                  <ActionList.LeadingVisual>
                    <PencilIcon />
                  </ActionList.LeadingVisual>
                  Edit name
                </ActionList.Item>
                <ActionList.Item variant="danger" onSelect={() => setConfirmDelete(true)}>
                  <ActionList.LeadingVisual>
                    <TrashIcon />
                  </ActionList.LeadingVisual>
                  Delete object
                </ActionList.Item>
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
        )}
      </Box>

      {editingName && (
        <SettingsPanel
          slug={slug}
          config={config}
          onClose={() => setEditingName(false)}
          onSaved={(c) => {
            setConfig(c);
            setConfigs((cs) => cs.map((x) => (x.id === c.id ? c : x)));
            setEditingName(false);
          }}
        />
      )}

      {/* File browser card */}
      <div className="otdm-fb">
        <FileTree slug={slug} projectName={project.name} configs={configs} activeId={configId} />
        <div className="otdm-fb-main">
          <div className="otdm-fb-toolbar">
            <span className="otdm-fb-crumb">
              <b>{config.name}</b>
            </span>
            {!split && <BranchEnvMenu value={env} envs={envs} onChange={setEnv} />}
            {!split && <DeltaBadge slug={slug} config={config} env={env} refreshToken={refresh} />}
            <Box sx={{ flex: 1 }} />
            {!split && !readOnly && (
              <Button leadingVisual={editing ? EyeIcon : PencilIcon} onClick={() => setEditing((v) => !v)}>
                {editing ? "View" : "Edit"}
              </Button>
            )}
            {!editing && isVar && (
              <Button leadingVisual={reveal ? EyeClosedIcon : EyeIcon} onClick={() => setReveal((v) => !v)}>
                {reveal ? "Hide" : "Reveal"}
              </Button>
            )}
            {!split && !editing && (
              <Button leadingVisual={CopyIcon} onClick={copy}>
                {copied ? "Copied" : "Copy"}
              </Button>
            )}
            <Button leadingVisual={ColumnsIcon} variant={split ? "primary" : "default"} onClick={toggleSplit}>
              {split ? "Exit split" : "Split"}
            </Button>
          </div>
          <div className="otdm-fb-body">
            {split ? (
              <SplitCompare
                slug={slug}
                config={config}
                envs={envs}
                panes={panes}
                setPanes={setPanes}
                reveal={reveal}
                refreshToken={refresh}
              />
            ) : editing ? (
              <Box sx={{ p: 3 }}>
                <EditorDispatch
                  key={`${config.id}:${env}:${refresh}`}
                  slug={slug}
                  config={config}
                  layer={env}
                  readOnly={readOnly}
                  onSaved={bump}
                />
              </Box>
            ) : (
              <CodeFileView
                slug={slug}
                config={config}
                env={env}
                reveal={reveal}
                refreshToken={refresh}
                onText={setCopyText}
              />
            )}
          </div>
        </div>
      </div>

      {/* Version history for the selected environment layer */}
      <Box>
        <Heading sx={{ fontSize: 2, mb: 2 }}>Version history</Heading>
        <VersionHistory
          key={`vh:${config.id}:${env}:${refresh}`}
          slug={slug}
          config={config}
          layer={env}
          onRolledBack={bump}
        />
      </Box>

      {confirmDelete && (
        <ConfirmationDialog
          title={`Delete “${config.name}”?`}
          confirmButtonContent="Delete object"
          confirmButtonType="danger"
          onClose={(gesture) => (gesture === "confirm" ? void remove() : setConfirmDelete(false))}
        >
          This archives the object across all environments — its values, file content, and version history stop resolving.
        </ConfirmationDialog>
      )}
    </Box>
  );
}

function SettingsPanel({
  slug,
  config,
  onClose,
  onSaved,
}: {
  slug: string;
  config: Config;
  onClose: () => void;
  onSaved: (c: Config) => void;
}) {
  const [name, setName] = useState(config.name);
  const [sortOrder, setSortOrder] = useState(String(config.sort_order));
  const [description, setDescription] = useState(config.description);
  const [err, setErr] = useState("");

  async function save(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      const updated = await api.updateConfig(slug, config.id, {
        name: name.trim() || config.name,
        sort_order: Number(sortOrder) || 0,
        description,
      });
      onSaved(updated);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box as="form" onSubmit={save} sx={{ p: 3, bg: "canvas.subtle", borderRadius: 2, display: "grid", gap: 2 }}>
      {err && <Flash variant="danger">{err}</Flash>}
      <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", alignItems: "flex-end" }}>
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} />
        </FormControl>
        <FormControl sx={{ width: 120 }}>
          <FormControl.Label>Sort order</FormControl.Label>
          <TextInput type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} />
        </FormControl>
      </Box>
      <FormControl>
        <FormControl.Label>Description</FormControl.Label>
        <TextInput block value={description} onChange={(e) => setDescription(e.target.value)} />
      </FormControl>
      <Box sx={{ display: "flex", gap: 2 }}>
        <Button type="submit" variant="primary">
          Save
        </Button>
        <Button type="button" onClick={onClose}>
          Cancel
        </Button>
      </Box>
    </Box>
  );
}
