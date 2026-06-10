import { useEffect, useState } from "react";
import { Box, Button, Checkbox, Flash, FormControl, Heading, Label, Spinner, Text } from "../../ui/primer";
import { AdminUser, api } from "../../api";
import { errMessage } from "../../lib/errors";
import { Avatar, Overline } from "../ui";

// Instance user directory (admin only). Moved from the old /users page into the
// consolidated Settings → Instance admin → Users panel.
export default function UsersPanel({ meId }: { meId?: string }) {
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [err, setErr] = useState("");

  async function load() {
    setErr("");
    try {
      setUsers(await api.listUsers());
    } catch (e) {
      setErr(errMessage(e));
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
    } catch (e) {
      setErr(errMessage(e));
    }
  }

  if (!users) return err ? <Flash variant="danger">{err}</Flash> : <Spinner />;

  return (
    <Box>
      <Overline>Instance admin</Overline>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Users</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Everyone with an account on this opentdm instance.
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
            <Avatar name={u.username} size={28} />
            <Box>
              <Text sx={{ fontWeight: "bold" }}>
                {u.username}
                {u.id === meId ? " (you)" : ""}
              </Text>
              <Text sx={{ color: "fg.muted", fontSize: 0, display: "block" }}>{u.email}</Text>
            </Box>
            {u.is_admin && <Label variant="accent">admin</Label>}
            {!u.is_active && <Label variant="danger">deactivated</Label>}
            <Box sx={{ flex: 1 }} />
            <FormControl>
              <Checkbox checked={u.is_admin} onChange={() => toggle(u, { is_admin: !u.is_admin })} />
              <FormControl.Label>Admin</FormControl.Label>
            </FormControl>
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
