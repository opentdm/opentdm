import { FormEvent, useState } from "react";
import { ShieldLockIcon } from "@primer/octicons-react";
import { Box, Button, Flash, FormControl, Heading, Label, Text, TextInput } from "../../ui/primer";
import { api, User } from "../../api";
import { useToast } from "../../lib/toast";
import Overline from "../Overline";

function initials(name: string): string {
  const parts = name.split(/[.\-_@\s]+/).filter(Boolean);
  const letters = parts.length >= 2 ? parts[0][0] + parts[1][0] : name.slice(0, 2);
  return letters.toUpperCase();
}

export default function ProfilePanel({ me }: { me: User }) {
  const toast = useToast();
  const [email, setEmail] = useState(me.email);
  const [savedEmail, setSavedEmail] = useState(me.email);
  const [emailErr, setEmailErr] = useState("");
  const [emailBusy, setEmailBusy] = useState(false);

  const [cur, setCur] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [pwBusy, setPwBusy] = useState(false);

  async function saveEmail(e: FormEvent) {
    e.preventDefault();
    setEmailErr("");
    setEmailBusy(true);
    try {
      await api.patch("/auth/me", { email: email.trim() });
      setSavedEmail(email.trim());
      toast("Email updated.");
    } catch (e) {
      setEmailErr(e instanceof Error ? e.message : "Failed to update email");
    } finally {
      setEmailBusy(false);
    }
  }

  async function changePassword(e: FormEvent) {
    e.preventDefault();
    setPwErr("");
    if (next.length < 8) {
      setPwErr("New password must be at least 8 characters.");
      return;
    }
    if (next !== confirm) {
      setPwErr("New password and confirmation don't match.");
      return;
    }
    setPwBusy(true);
    try {
      await api.post("/auth/change-password", { current_password: cur, new_password: next });
      setCur("");
      setNext("");
      setConfirm("");
      toast("Password changed.");
    } catch (e) {
      setPwErr(e instanceof Error ? e.message : "Failed to change password");
    } finally {
      setPwBusy(false);
    }
  }

  return (
    <Box>
      <Overline>Account</Overline>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Profile</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Your account identity. Username is fixed — it's used in audit logs and login.
      </Text>

      <Box sx={{ display: "flex", gap: 3, alignItems: "center", mb: 4 }}>
        <span className="otdm-avatar-lg" aria-hidden="true">
          {initials(me.username)}
        </span>
        <Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
            <Text sx={{ fontWeight: "bold", fontSize: 2 }}>{me.username}</Text>
            {me.is_admin && <Label variant="accent">admin</Label>}
          </Box>
          <Text sx={{ color: "fg.muted", display: "block" }}>{savedEmail}</Text>
        </Box>
      </Box>

      <Flash sx={{ mb: 4 }}>
        <ShieldLockIcon /> Profile is managed by your instance admin.
      </Flash>

      <Box as="form" onSubmit={saveEmail} sx={{ display: "grid", gap: 2, maxWidth: 420, mb: 4 }}>
        {emailErr && <Flash variant="danger">{emailErr}</Flash>}
        <FormControl disabled>
          <FormControl.Label>Username</FormControl.Label>
          <TextInput block value={me.username} disabled />
        </FormControl>
        <FormControl>
          <FormControl.Label>Email</FormControl.Label>
          <TextInput block type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </FormControl>
        <Box>
          <Button type="submit" variant="primary" disabled={emailBusy || !email.trim() || email.trim() === savedEmail}>
            {emailBusy ? "Saving…" : "Save email"}
          </Button>
        </Box>
      </Box>

      <Heading sx={{ fontSize: 2, mb: 2 }}>Change password</Heading>
      <Box as="form" onSubmit={changePassword} sx={{ display: "grid", gap: 2, maxWidth: 420 }}>
        {pwErr && <Flash variant="danger">{pwErr}</Flash>}
        <FormControl>
          <FormControl.Label>Current password</FormControl.Label>
          <TextInput
            block
            type="password"
            value={cur}
            onChange={(e) => setCur(e.target.value)}
            autoComplete="current-password"
          />
        </FormControl>
        <FormControl>
          <FormControl.Label>New password</FormControl.Label>
          <TextInput
            block
            type="password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            autoComplete="new-password"
          />
          <FormControl.Caption>At least 8 characters.</FormControl.Caption>
        </FormControl>
        <FormControl>
          <FormControl.Label>Confirm new password</FormControl.Label>
          <TextInput
            block
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
          />
        </FormControl>
        <Box>
          <Button type="submit" variant="primary" disabled={pwBusy || !cur || !next || !confirm}>
            {pwBusy ? "Changing…" : "Change password"}
          </Button>
        </Box>
      </Box>
    </Box>
  );
}
