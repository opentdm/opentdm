import { useEffect, useState } from "react";
import { Box, Button, Flash, Label, Text } from "@primer/react";
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
      <Text sx={{ fontWeight: "bold", display: "block", mb: 2 }}>History — {layer}</Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {versions.length === 0 && <Text sx={{ color: "fg.muted" }}>No versions yet.</Text>}
      {versions.map((v) => (
        <Box key={v.version} sx={{ display: "flex", alignItems: "center", gap: 2, py: 1 }}>
          <Text sx={{ fontFamily: "mono" }}>v{v.version}</Text>
          {v.is_current && <Label variant="success">current</Label>}
          <Text sx={{ color: "fg.muted", fontSize: 0 }}>{new Date(v.created_at).toLocaleString()}</Text>
          {v.comment && <Text sx={{ color: "fg.muted", fontSize: 0 }}>· {v.comment}</Text>}
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
      {diff && (
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
