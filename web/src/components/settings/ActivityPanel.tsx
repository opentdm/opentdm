import { useCallback } from "react";
import { Box, Heading, Text } from "../../ui/primer";
import { api } from "../../api";
import AuditFeed from "../AuditFeed";

// Instance-wide activity (admin only). Reuses the keyset-paginated AuditFeed.
export default function ActivityPanel() {
  const load = useCallback((before?: string) => api.listAudit(before), []);
  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Activity</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>Resource changes across all projects.</Text>
      <AuditFeed load={load} />
    </Box>
  );
}
