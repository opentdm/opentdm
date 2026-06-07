import { useCallback } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { Box, Breadcrumbs, Heading, Text } from "@primer/react";
import { api } from "../api";
import AuditFeed from "../components/AuditFeed";

// Activity feed. With a :slug it's the project feed (members can view); without,
// it's the instance-wide admin feed (route is admin-gated).
export default function Activity() {
  const { slug } = useParams();
  const load = useCallback(
    (before?: string) => (slug ? api.listProjectAudit(slug, before) : api.listAudit(before)),
    [slug],
  );

  return (
    <Box sx={{ display: "grid", gap: 3 }}>
      {slug ? (
        <Box>
          <Breadcrumbs>
            <Breadcrumbs.Item as={RouterLink} to={`/projects/${slug}`}>
              {slug}
            </Breadcrumbs.Item>
            <Breadcrumbs.Item selected>Activity</Breadcrumbs.Item>
          </Breadcrumbs>
          <Heading sx={{ fontSize: 4, mt: 2 }}>Activity</Heading>
        </Box>
      ) : (
        <Box>
          <Heading sx={{ fontSize: 4 }}>Activity</Heading>
          <Text sx={{ color: "fg.muted" }}>Resource changes across all projects.</Text>
        </Box>
      )}
      <AuditFeed key={slug ?? "global"} load={load} />
    </Box>
  );
}
