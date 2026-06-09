import { useEffect, useState } from "react";
import { ChevronDownIcon, ChevronUpIcon, HistoryIcon } from "@primer/octicons-react";
import { Box, Button, Flash, Label, Text } from "../ui/primer";
import { api, Config, DiffResult, VersionMeta } from "../api";

interface VersionHistoryProps {
  slug: string;
  config: Config;
  layer: string;
  onRolledBack: () => void;
}

export default function VersionHistory({ slug, config, layer, onRolledBack }: VersionHistoryProps) {
  const [versions, setVersions] = useState<VersionMeta[]>([]);
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [err, setErr] = useState("");
  const [open, setOpen] = useState(false);

  async function load() {
    try {
      setVersions(await api.get<VersionMeta[]>(`/projects/${slug}/configs/${config.id}/versions?env=${layer}`));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    setDiff(null);
    void load();
  }, [config.id, layer]);

  async function showDiff(v: number) {
    setErr("");
    try {
      setDiff(await api.get<DiffResult>(`/projects/${slug}/configs/${config.id}/diff?env=${layer}&from=${v}&to=0`));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function rollback(v: number) {
    setErr("");
    try {
      await api.post(`/projects/${slug}/configs/${config.id}/rollback`, { env: layer, to_version: v });
      await load();
      onRolledBack();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box sx={{ mt: 3, p: 3, bg: "canvas.subtle", borderRadius: 2 }}>
      <Box sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 2 }}>
        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
          <HistoryIcon />
          <Text sx={{ fontWeight: "bold" }}>Version history</Text>
        </Box>
        <Button size="small" variant="invisible" onClick={() => setOpen((o) => !o)}>
          {open ? "Hide" : "View"} {open ? <ChevronUpIcon /> : <ChevronDownIcon />}
        </Button>
      </Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {open && versions.length === 0 && <Text sx={{ color: "fg.muted" }}>No versions yet.</Text>}
      {open &&
        versions.map((v) => (
          <Box key={v.version} sx={{ display: "flex", alignItems: "center", gap: 2, py: 1 }}>
            <Text sx={{ fontFamily: "mono", color: "fg.accent" }}>v{v.version}</Text>
            {v.is_current && <Label className="otdm-pill-accent">current</Label>}
            <Text sx={{ color: "fg.muted", fontSize: 0 }}>{new Date(v.created_at).toLocaleString()}</Text>
            {v.comment && <Text sx={{ color: "fg.muted", fontSize: 0 }}>· {v.comment}</Text>}
            {v.added ? <Text sx={{ color: "fg.muted", fontSize: 0 }}>+{v.added}</Text> : null}
            {v.changed ? <Text sx={{ color: "fg.muted", fontSize: 0 }}>~{v.changed}</Text> : null}
            {v.removed ? <Text sx={{ color: "fg.muted", fontSize: 0 }}>-{v.removed}</Text> : null}
            <Box sx={{ flex: 1 }} />
            <Button size="small" variant="invisible" onClick={() => showDiff(v.version)}>
              Diff vs current
            </Button>
            {!v.is_current && (
              <Button size="small" onClick={() => rollback(v.version)}>
                Rollback
              </Button>
            )}
          </Box>
        ))}
      {open && diff && (
        <Box sx={{ mt: 2 }}>
          <Text sx={{ fontWeight: "bold", display: "block", mb: 1 }}>
            Diff v{diff.from} → v{diff.to}
          </Text>
          {diff.kind === "variable" ? (
            (diff.vars || []).length === 0 ? (
              <Text sx={{ color: "fg.muted" }}>No differences.</Text>
            ) : (
              (diff.vars || []).map((d) => (
                <Box key={d.key} sx={{ fontFamily: "mono", fontSize: 0 }}>
                  <Label
                    sx={{ mr: 2 }}
                    variant={d.status === "added" ? "success" : d.status === "removed" ? "danger" : "attention"}
                  >
                    {d.status}
                  </Label>
                  {d.key}: {d.from ?? "∅"} → {d.to ?? "∅"}
                </Box>
              ))
            )
          ) : (
            <Box as="pre" sx={{ fontFamily: "mono", fontSize: 0, whiteSpace: "pre-wrap", overflow: "auto" }}>
              {diff.file_diff || "(no changes)"}
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
}
