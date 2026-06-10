import { FormEvent, useState } from "react";
import { Box, Button, FormControl, Flash, Heading, Text, TextInput } from "../ui/primer";
import { api } from "../api";
import { errMessage } from "../lib/errors";

export default function Setup({ onDone }: { onDone: () => void }) {
  const [setupToken, setSetupToken] = useState("");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/auth/bootstrap", {
        setup_token: setupToken,
        username,
        email,
        password,
      });
      onDone();
    } catch (e) {
      setErr(errMessage(e) || "Setup failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Box sx={{ maxWidth: 380, mx: "auto", pt: 6 }}>
      <Heading sx={{ mb: 1, fontSize: 4 }}>Welcome to opentdm</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Create the first admin. Paste the setup token printed in the server logs.
      </Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box as="form" onSubmit={submit} sx={{ display: "grid", gap: 3 }}>
        <FormControl>
          <FormControl.Label>Setup token</FormControl.Label>
          <TextInput block value={setupToken} onChange={(e) => setSetupToken(e.target.value)} />
        </FormControl>
        <FormControl>
          <FormControl.Label>Username</FormControl.Label>
          <TextInput block value={username} onChange={(e) => setUsername(e.target.value)} />
        </FormControl>
        <FormControl>
          <FormControl.Label>Email</FormControl.Label>
          <TextInput block type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </FormControl>
        <FormControl>
          <FormControl.Label>Password</FormControl.Label>
          <TextInput block type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </FormControl>
        <Button type="submit" variant="primary" disabled={busy} block>
          Create admin
        </Button>
      </Box>
    </Box>
  );
}
