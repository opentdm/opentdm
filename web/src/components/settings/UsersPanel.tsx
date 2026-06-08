import { useEffect, useState } from "react";
import { Box, Button, Flash, Heading, Label, Spinner, Text } from "../../ui/primer";
import { AdminUser, api } from "../../api";

// Instance user directory (admin only). Moved from the old /users page into the
// consolidated Settings → Instance admin → Users panel.
export default function UsersPanel() {
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [err, setErr] = useState("");

  async function load() {
    setErr("");
    try {
      setUsers(await api.listUsers());
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  async function toggle(u: AdminUser, patch: { is_active?: boolean; is_admin?: boolean }) {
    setErr("");
    try {
      await api.updateUser(u.id, patch);
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  if (!users) return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Users</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Instance users. New users join via project invitations; here you can grant admin or deactivate accounts.
      </Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {users.map((u, i) => (
          <Box
            key={u.id}
            sx={{
              p: 3,
              display: "flex",
              alignItems: "center",
              gap: 2,
              flexWrap: "wrap",
              borderBottomWidth: i < users.length - 1 ? 1 : 0,
              borderBottomStyle: "solid",
              borderColor: "border.muted",
            }}
          >
            <Box>
              <Text sx={{ fontWeight: "bold" }}>{u.username}</Text>
              <Text sx={{ color: "fg.muted", fontSize: 0, display: "block" }}>{u.email}</Text>
            </Box>
            {u.is_admin && <Label variant="accent">admin</Label>}
            {!u.is_active && <Label variant="danger">deactivated</Label>}
            <Box sx={{ flex: 1 }} />
            <Button size="small" onClick={() => toggle(u, { is_admin: !u.is_admin })}>
              {u.is_admin ? "Revoke admin" : "Make admin"}
            </Button>
            <Button
              size="small"
              variant={u.is_active ? "danger" : "default"}
              onClick={() => toggle(u, { is_active: !u.is_active })}
            >
              {u.is_active ? "Deactivate" : "Activate"}
            </Button>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
