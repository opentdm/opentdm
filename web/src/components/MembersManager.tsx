import { FormEvent, useEffect, useState } from "react";
import { Box, Button, Flash, FormControl, Heading, IconButton, Label, Select, Text, TextInput } from "../ui/primer";
import { TrashIcon } from "@primer/octicons-react";
import { api, canManage, Invitation, Member } from "../api";

const ROLES = ["viewer", "editor", "owner"];

interface MembersManagerProps {
  slug: string;
  role?: string; // caller's role on the project (owners manage)
}

// Lists a project's members; owners (and admins) can add an existing user by
// username/email, change roles, and remove members. Everyone else sees a
// read-only list. The server enforces the keep-≥1-owner guard (surfaced here).
export default function MembersManager({ slug, role }: MembersManagerProps) {
  const manage = canManage(role);
  const [members, setMembers] = useState<Member[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [user, setUser] = useState("");
  const [addRole, setAddRole] = useState("viewer");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("viewer");
  const [inviteLink, setInviteLink] = useState("");
  const [err, setErr] = useState("");

  async function load() {
    setErr("");
    try {
      setMembers(await api.listMembers(slug));
      if (manage) setInvitations(await api.listInvitations(slug));
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function invite(e: FormEvent) {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    setErr("");
    setInviteLink("");
    try {
      const res = await api.createInvitation(slug, { email: inviteEmail.trim(), role: inviteRole });
      setInviteEmail("");
      if (!res.email_sent && res.accept_url) setInviteLink(res.accept_url);
      await load();
    } catch (e: any) {
      setErr(e.message);
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
    if (!user.trim()) return;
    await run(async () => {
      await api.addMember(slug, { user: user.trim(), role: addRole });
      setUser("");
    });
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Members</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Who can access this project. Owners manage members; editors read + write; viewers read only.
      </Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2, mb: manage ? 3 : 0 }}>
        {members.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No members.</Box>}
        {members.map((m, i) => (
          <Box
            key={m.user_id}
            sx={{
              p: 3,
              display: "flex",
              alignItems: "center",
              gap: 2,
              borderBottomWidth: i < members.length - 1 ? 1 : 0,
              borderBottomStyle: "solid",
              borderColor: "border.muted",
            }}
          >
            <Box>
              <Text sx={{ fontWeight: "bold" }}>{m.username}</Text>
              <Text sx={{ color: "fg.muted", fontSize: 0, display: "block" }}>{m.email}</Text>
            </Box>
            <Box sx={{ flex: 1 }} />
            {manage ? (
              <Select value={m.role} onChange={(e) => run(() => api.updateMember(slug, m.user_id, e.target.value))}>
                {ROLES.map((r) => (
                  <Select.Option key={r} value={r}>
                    {r}
                  </Select.Option>
                ))}
              </Select>
            ) : (
              <Label>{m.role}</Label>
            )}
            {manage && (
              <IconButton
                icon={TrashIcon}
                aria-label="remove member"
                size="small"
                variant="danger"
                onClick={() => run(() => api.removeMember(slug, m.user_id))}
              />
            )}
          </Box>
        ))}
      </Box>
      {manage && (
        <Box as="form" onSubmit={add} sx={{ display: "flex", gap: 2, alignItems: "flex-end", flexWrap: "wrap" }}>
          <FormControl>
            <FormControl.Label>Add member (username or email)</FormControl.Label>
            <TextInput value={user} onChange={(e) => setUser(e.target.value)} placeholder="bob" />
          </FormControl>
          <FormControl>
            <FormControl.Label>Role</FormControl.Label>
            <Select value={addRole} onChange={(e) => setAddRole(e.target.value)}>
              {ROLES.map((r) => (
                <Select.Option key={r} value={r}>
                  {r}
                </Select.Option>
              ))}
            </Select>
          </FormControl>
          <Button type="submit" variant="primary">
            Add
          </Button>
        </Box>
      )}

      {manage && (
        <Box sx={{ mt: 4 }}>
          <Heading sx={{ fontSize: 2, mb: 1 }}>Invite by email</Heading>
          <Text sx={{ color: "fg.muted", display: "block", mb: 2 }}>
            Invite someone who doesn't have an account yet. They set their own password on accept.
          </Text>
          {inviteLink && (
            <Flash variant="success" sx={{ mb: 2 }}>
              Email isn't configured — share this one-time accept link:
              <Box as="code" sx={{ display: "block", mt: 1, fontFamily: "mono", wordBreak: "break-all" }}>
                {inviteLink}
              </Box>
            </Flash>
          )}
          <Box as="form" onSubmit={invite} sx={{ display: "flex", gap: 2, alignItems: "flex-end", flexWrap: "wrap", mb: 3 }}>
            <FormControl>
              <FormControl.Label>Email</FormControl.Label>
              <TextInput
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="teammate@example.com"
              />
            </FormControl>
            <FormControl>
              <FormControl.Label>Role</FormControl.Label>
              <Select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)}>
                {ROLES.map((r) => (
                  <Select.Option key={r} value={r}>
                    {r}
                  </Select.Option>
                ))}
              </Select>
            </FormControl>
            <Button type="submit" variant="primary">
              Send invite
            </Button>
          </Box>
          {invitations.length > 0 && (
            <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
              {invitations.map((inv, i) => (
                <Box
                  key={inv.id}
                  sx={{
                    p: 3,
                    display: "flex",
                    alignItems: "center",
                    gap: 2,
                    borderBottomWidth: i < invitations.length - 1 ? 1 : 0,
                    borderBottomStyle: "solid",
                    borderColor: "border.muted",
                  }}
                >
                  <Text>{inv.email}</Text>
                  <Label>{inv.role}</Label>
                  <Text sx={{ color: "fg.muted", fontSize: 0 }}>pending</Text>
                  <Box sx={{ flex: 1 }} />
                  <IconButton
                    icon={TrashIcon}
                    aria-label="revoke invitation"
                    size="small"
                    variant="danger"
                    onClick={() => run(() => api.revokeInvitation(slug, inv.id))}
                  />
                </Box>
              ))}
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
}
