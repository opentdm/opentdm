import { FormEvent, useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Box, Button, Flash, FormControl, Heading, Spinner, Text, TextInput } from "../ui/primer";
import { api, InvitationInfo } from "../api";

// Public page reached from an invitation link. Shows the project + role, lets the
// invitee pick a username/password, then creates the account + membership and
// logs them in.
export default function AcceptInvite({ onDone }: { onDone: () => void | Promise<void> }) {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const nav = useNavigate();
  const [info, setInfo] = useState<InvitationInfo | null>(null);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!token) {
      setErr("Missing invitation token.");
      setLoading(false);
      return;
    }
    api
      .getInvitation(token)
      .then(setInfo)
      .catch(() => setErr("This invitation is invalid, expired, or already used."))
      .finally(() => setLoading(false));
  }, [token]);

  async function accept(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      await api.acceptInvitation(token, { username, password });
      await onDone(); // refresh auth state and WAIT for `me` before navigating
      nav(`/projects/${info?.project_slug ?? ""}`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", pt: 6 }}>
        <Spinner />
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 420, mx: "auto", pt: 6, px: 3 }}>
      <Heading sx={{ fontSize: 4, mb: 2 }}>Accept invitation</Heading>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      {info && (
        <>
          <Text sx={{ display: "block", mb: 3, color: "fg.muted" }}>
            You've been invited to <b>{info.project}</b> as <b>{info.role}</b> ({info.email}). Choose a username and
            password to create your account.
          </Text>
          <Box as="form" onSubmit={accept} sx={{ display: "grid", gap: 3 }}>
            <FormControl>
              <FormControl.Label>Username</FormControl.Label>
              <TextInput block value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
            </FormControl>
            <FormControl>
              <FormControl.Label>Password</FormControl.Label>
              <TextInput
                block
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
              />
              <FormControl.Caption>At least 8 characters.</FormControl.Caption>
            </FormControl>
            <Button type="submit" variant="primary">
              Accept &amp; create account
            </Button>
          </Box>
        </>
      )}
    </Box>
  );
}
