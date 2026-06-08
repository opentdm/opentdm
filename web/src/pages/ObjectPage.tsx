import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink, useNavigate, useParams } from "react-router-dom";
import {
  ActionList,
  ActionMenu,
  Box,
  Breadcrumbs,
  Button,
  ConfirmationDialog,
  Dialog,
  Flash,
  FormControl,
  Heading,
  Label,
  Spinner,
  Text,
  TextInput,
  Token,
} from "../ui/primer";
import { EyeIcon, GearIcon, KebabHorizontalIcon, PencilIcon, TrashIcon } from "@primer/octicons-react";
import { api, canWrite, Config, Environment } from "../api";
import EditorDispatch from "../components/editors/EditorDispatch";
import ResolvedView from "../components/ResolvedView";
import VersionHistory from "../components/VersionHistory";

export default function ObjectPage() {
  const { slug = "", configId = "" } = useParams();
  const nav = useNavigate();
  const [config, setConfig] = useState<Config | null>(null);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [role, setRole] = useState<string | undefined>(undefined);
  const [layer, setLayer] = useState("base");
  const [editing, setEditing] = useState(false);
  const [resolveOpen, setResolveOpen] = useState(false);
  const [showResolved, setShowResolved] = useState(false);
  const [reloadNonce, setReloadNonce] = useState(0);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [err, setErr] = useState("");

  async function load() {
    try {
      const [c, e, p] = await Promise.all([api.getConfig(slug, configId), api.listEnvs(slug), api.getProject(slug)]);
      setConfig(c);
      setEnvs(e);
      setRole(p.your_role);
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
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

  if (!config) {
    return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;
  }

  const layers = ["base", ...envs.map((e) => e.slug)];
  const readOnly = !canWrite(role);

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

      <Box sx={{ display: "flex", alignItems: "center", gap: 2, flexWrap: "wrap" }}>
        <Heading sx={{ fontSize: 4 }}>{config.name}</Heading>
        <Label variant="secondary">{config.format}</Label>
        <Label variant={config.kind === "file" ? "accent" : "default"}>{config.kind}</Label>
        {config.tags.map((t) => (
          <Token key={t} text={t} />
        ))}
        <Box sx={{ flex: 1 }} />
        {(config.kind === "variable" || !readOnly) && (
          <ActionMenu>
            <ActionMenu.Anchor>
              <Button leadingVisual={KebabHorizontalIcon}>Object</Button>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay width="small">
              <ActionList>
                {config.kind === "variable" && (
                  <ActionList.Item onSelect={() => setResolveOpen(true)}>
                    <ActionList.LeadingVisual>
                      <EyeIcon />
                    </ActionList.LeadingVisual>
                    View resolved
                  </ActionList.Item>
                )}
                {!readOnly && (
                  <ActionList.Item onSelect={() => setEditing((v) => !v)}>
                    <ActionList.LeadingVisual>
                      <PencilIcon />
                    </ActionList.LeadingVisual>
                    Edit name & tags
                  </ActionList.Item>
                )}
                {!readOnly && (
                  <ActionList.Item variant="danger" onSelect={() => setConfirmDelete(true)}>
                    <ActionList.LeadingVisual>
                      <TrashIcon />
                    </ActionList.LeadingVisual>
                    Delete object
                  </ActionList.Item>
                )}
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
        )}
      </Box>

      {editing && (
        <SettingsPanel
          slug={slug}
          config={config}
          onClose={() => setEditing(false)}
          onSaved={(c) => {
            setConfig(c);
            setEditing(false);
          }}
        />
      )}

      {/* Environment switcher — base is the shared layer; each env overrides it. */}
      <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap", borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.default", pb: 2 }}>
        {layers.map((l) => {
          const env = envs.find((e) => e.slug === l);
          return (
            <Button
              key={l}
              size="small"
              variant={layer === l ? "primary" : "invisible"}
              onClick={() => setLayer(l)}
              trailingVisual={env?.is_default ? () => <Label variant="accent">default</Label> : undefined}
            >
              {l}
            </Button>
          );
        })}
      </Box>

      <EditorDispatch
        key={`${config.id}:${layer}:${reloadNonce}`}
        slug={slug}
        config={config}
        layer={layer}
        readOnly={readOnly}
      />

      {config.kind === "variable" && (
        <Box>
          <Button variant="invisible" leadingVisual={EyeIcon} onClick={() => setShowResolved((v) => !v)}>
            {showResolved ? "Hide resolved" : "Show resolved"}
          </Button>
          {showResolved && (
            <Box sx={{ mt: 2 }}>
              <ResolvedView slug={slug} config={config} envs={envs} initialEnv={layer === "base" ? undefined : layer} />
            </Box>
          )}
        </Box>
      )}

      <Box>
        <Button variant="invisible" leadingVisual={GearIcon} onClick={() => setShowHistory((v) => !v)}>
          {showHistory ? "Hide history" : "History & rollback"}
        </Button>
        {showHistory && (
          <VersionHistory
            key={`vh:${config.id}:${layer}:${reloadNonce}`}
            slug={slug}
            config={config}
            layer={layer}
            onRolledBack={() => setReloadNonce((n) => n + 1)}
          />
        )}
      </Box>

      {resolveOpen && (
        <Dialog title={`Resolved — ${config.name}`} width="large" onClose={() => setResolveOpen(false)}>
          <ResolvedView slug={slug} config={config} envs={envs} initialEnv={layer === "base" ? undefined : layer} />
        </Dialog>
      )}

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
  const [tags, setTags] = useState(config.tags.join(", "));
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
        tags: tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
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
        <FormControl sx={{ flex: 1, minWidth: 200 }}>
          <FormControl.Label>Tags</FormControl.Label>
          <TextInput block value={tags} onChange={(e) => setTags(e.target.value)} placeholder="prod, payments" />
          <FormControl.Caption>Comma-separated.</FormControl.Caption>
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
