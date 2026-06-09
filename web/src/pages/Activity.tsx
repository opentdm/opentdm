import { useCallback } from "react";
import { useParams } from "react-router-dom";
import { Box, Heading } from "../ui/primer";
import { api } from "../api";
import AuditFeed from "../components/AuditFeed";

// Activity feed. With a :slug it's the project feed (members can view); the
// breadcrumb above (global Topbar) shows the project context.
export default function Activity() {
  const { slug } = useParams();
  const load = useCallback(
    (before?: string) => (slug ? api.listProjectAudit(slug, before) : api.listAudit(before)),
    [slug],
  );

  return (
    <Box sx={{ display: "grid", gap: 3 }}>
      <Heading sx={{ fontSize: 4 }}>Activity</Heading>
      <AuditFeed key={slug ?? "global"} load={load} />
    </Box>
  );
}
