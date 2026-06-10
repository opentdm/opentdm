import { useCallback } from "react";
import { Box, Heading, Text } from "../../ui/primer";
import { api } from "../../api";
import AuditFeed from "../AuditFeed";
import { Overline } from "../ui";

// Instance-wide activity (admin only). Reuses the keyset-paginated AuditFeed.
export default function ActivityPanel() {
  const load = useCallback((before?: string) => api.listAudit(before), []);
  return (
    <Box>
      <Overline>Instance</Overline>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Activity</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Append-only audit feed across all projects. No secret values are recorded.
      </Text>
      <AuditFeed load={load} />
    </Box>
  );
}
