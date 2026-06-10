import { useEffect, useState } from "react";
import { Box, Button, Flash, Label, Spinner, Text } from "../ui/primer";
import { AuditEntry, auditLabel } from "../api";
import { errMessage } from "../lib/errors";
import Avatar from "./Avatar";

function statusVariant(status: number): "success" | "attention" | "danger" | "secondary" {
  if (status >= 200 && status < 300) return "success";
  if (status >= 300 && status < 400) return "attention";
  if (status >= 400) return "danger";
  return "secondary";
}

function relativeTime(iso: string): string {
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return new Date(iso).toLocaleDateString();
}

interface AuditFeedProps {
  load: (before?: string) => Promise<{ entries: AuditEntry[]; next?: string }>;
}

// A keyset-paginated activity feed. The parent should key it per feed (e.g. by
// project slug) so switching feeds remounts with a fresh first page.
export default function AuditFeed({ load }: AuditFeedProps) {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [next, setNext] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [paging, setPaging] = useState(false);
  const [err, setErr] = useState("");

  async function page(before?: string) {
    if (paging) return; // guard against double "Load more" clicks
    setPaging(true);
    setErr("");
    try {
      const res = await load(before);
      setEntries((prev) => (before ? [...prev, ...res.entries] : res.entries));
      setNext(res.next);
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setPaging(false);
      setLoading(false);
    }
  }
  useEffect(() => {
    setLoading(true);
    void page(undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading) return <Spinner />;
  return (
    <Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      {entries.length === 0 && <Text sx={{ color: "fg.muted" }}>No activity yet.</Text>}
      {entries.length > 0 && (
        <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
          {entries.map((e, i) => (
            <Box
              key={e.id}
              sx={{
                p: 3,
                display: "flex",
                alignItems: "center",
                gap: 2,
                flexWrap: "wrap",
                borderBottomWidth: i < entries.length - 1 ? 1 : 0,
                borderBottomStyle: "solid",
                borderColor: "border.muted",
              }}
            >
              <Avatar name={e.actor} size={26} />
              <Text sx={{ fontWeight: "bold" }}>{e.actor || "unknown"}</Text>
              <Text>{auditLabel(e.action)}</Text>
              {e.target_type && <Label variant="secondary">{e.target_type}</Label>}
              <Box sx={{ flex: 1 }} />
              <Text sx={{ color: "fg.muted", fontSize: 0 }} title={new Date(e.created_at).toLocaleString()}>
                {relativeTime(e.created_at)}
              </Text>
              <Label variant={statusVariant(e.status)}>{e.status}</Label>
            </Box>
          ))}
        </Box>
      )}
      {next && (
        <Box sx={{ mt: 3 }}>
          <Button onClick={() => page(next)} disabled={paging}>
            Load more
          </Button>
        </Box>
      )}
    </Box>
  );
}
