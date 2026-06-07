import { FormEvent, useEffect, useState } from "react";
import {
  Box,
  Button,
  ConfirmationDialog,
  Flash,
  FormControl,
  Heading,
  IconButton,
  Label,
  Text,
  TextInput,
} from "../ui/primer";
import { ChevronDownIcon, ChevronUpIcon, PencilIcon, TrashIcon } from "@primer/octicons-react";
import { api, Environment } from "../api";

interface EnvironmentManagerProps {
  slug: string;
}

// Manages a project's ordered, named environment layers: add / rename / reorder
// / set-default / delete. IDs are stable; slugs are renameable (consumers that
// reference a slug — /resolve, CLI, Actions — must be updated on rename).
export default function EnvironmentManager({ slug }: EnvironmentManagerProps) {
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [editId, setEditId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<Environment | null>(null);
  const [newName, setNewName] = useState("");

  async function load() {
    setErr("");
    try {
      setEnvs(await api.listEnvs(slug));
    } catch (e: any) {
      setErr(e.message);
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void load();
  }, [slug]);

  async function run(fn: () => Promise<unknown>) {
    setErr("");
    try {
      await fn();
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function add(e: FormEvent) {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    await run(async () => {
      await api.createEnv(slug, name);
      setNewName("");
    });
  }

  function move(index: number, dir: -1 | 1) {
    const target = index + dir;
    if (target < 0 || target >= envs.length) return;
    const ids = envs.map((e) => e.id);
    [ids[index], ids[target]] = [ids[target], ids[index]];
    void run(() => api.reorderEnvs(slug, ids));
  }

  if (loading) return <Text sx={{ color: "fg.muted" }}>Loading environments…</Text>;

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Environments</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Ordered layers resolved on top of <b>base</b>. Rename, reorder, or set which one is the default.
      </Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}

      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2, mb: 3 }}>
        {envs.map((env, i) =>
          editId === env.id ? (
            <EnvEditRow
              key={env.id}
              env={env}
              onCancel={() => setEditId(null)}
              onSave={async (patch) => {
                await run(() => api.updateEnv(slug, env.id, patch));
                setEditId(null);
              }}
            />
          ) : (
            <Box
              key={env.id}
              sx={{
                p: 3,
                display: "flex",
                alignItems: "center",
                gap: 2,
                borderBottomWidth: i < envs.length - 1 ? 1 : 0,
                borderBottomStyle: "solid",
                borderColor: "border.muted",
              }}
            >
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                <IconButton
                  icon={ChevronUpIcon}
                  aria-label="move up"
                  size="small"
                  variant="invisible"
                  disabled={i === 0}
                  onClick={() => move(i, -1)}
                />
                <IconButton
                  icon={ChevronDownIcon}
                  aria-label="move down"
                  size="small"
                  variant="invisible"
                  disabled={i === envs.length - 1}
                  onClick={() => move(i, 1)}
                />
              </Box>
              <Box>
                <Text sx={{ fontWeight: "bold" }}>{env.name}</Text>
                <Text sx={{ fontFamily: "mono", color: "fg.muted", fontSize: 0, display: "block" }}>{env.slug}</Text>
              </Box>
              {env.is_default && <Label variant="accent">default</Label>}
              <Box sx={{ flex: 1 }} />
              {!env.is_default && (
                <Button size="small" onClick={() => run(() => api.updateEnv(slug, env.id, { is_default: true }))}>
                  Set default
                </Button>
              )}
              <IconButton icon={PencilIcon} aria-label="rename" size="small" onClick={() => setEditId(env.id)} />
              <IconButton
                icon={TrashIcon}
                aria-label="delete"
                size="small"
                variant="danger"
                disabled={envs.length === 1}
                onClick={() => setDeleting(env)}
              />
            </Box>
          ),
        )}
      </Box>

      <Box as="form" onSubmit={add} sx={{ display: "flex", gap: 2, alignItems: "flex-end" }}>
        <FormControl>
          <FormControl.Label>New environment</FormControl.Label>
          <TextInput value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="qa" />
        </FormControl>
        <Button type="submit" variant="primary">
          Add
        </Button>
      </Box>

      {deleting && (
        <ConfirmationDialog
          title={`Delete “${deleting.name}”?`}
          confirmButtonContent="Delete environment"
          confirmButtonType="danger"
          onClose={(gesture) => {
            const env = deleting;
            setDeleting(null);
            if (gesture === "confirm" && env) void run(() => api.deleteEnv(slug, env.id));
          }}
        >
          This permanently removes all values, file content, and version history in <b>{deleting.slug}</b>, and any service
          tokens scoped to it lose access. This cannot be undone.
        </ConfirmationDialog>
      )}
    </Box>
  );
}

function EnvEditRow({
  env,
  onSave,
  onCancel,
}: {
  env: Environment;
  onSave: (patch: { slug: string; name: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(env.name);
  const [slug, setSlug] = useState(env.slug);

  return (
    <Box sx={{ p: 3, display: "flex", gap: 2, alignItems: "flex-end", flexWrap: "wrap", bg: "canvas.subtle" }}>
      <FormControl>
        <FormControl.Label>Name</FormControl.Label>
        <TextInput value={name} onChange={(e) => setName(e.target.value)} />
      </FormControl>
      <FormControl>
        <FormControl.Label>Slug</FormControl.Label>
        <TextInput value={slug} onChange={(e) => setSlug(e.target.value)} sx={{ fontFamily: "mono" }} />
        <FormControl.Caption>Renaming breaks consumers that reference this slug.</FormControl.Caption>
      </FormControl>
      <Button variant="primary" onClick={() => void onSave({ name: name.trim() || env.name, slug: slug.trim() || env.slug })}>
        Save
      </Button>
      <Button onClick={onCancel}>Cancel</Button>
    </Box>
  );
}
