import { FormEvent, useState } from "react";
import { Box, Button, FormControl, Flash, Heading, TextInput } from "@primer/react";
import { api } from "../api";

export default function Login({ onDone }: { onDone: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/auth/login", { username, password });
      onDone();
    } catch (e: any) {
      setErr(e.status === 401 ? "Invalid username or password" : e.message || "Login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Box sx={{ maxWidth: 340, mx: "auto", pt: 6 }}>
      <Heading sx={{ mb: 3, fontSize: 4 }}>Sign in to opentdm</Heading>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box as="form" onSubmit={submit} sx={{ display: "grid", gap: 3 }}>
        <FormControl>
          <FormControl.Label>Username</FormControl.Label>
          <TextInput block value={username} onChange={(e) => setUsername(e.target.value)} />
        </FormControl>
        <FormControl>
          <FormControl.Label>Password</FormControl.Label>
          <TextInput block type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </FormControl>
        <Button type="submit" variant="primary" disabled={busy} block>
          Sign in
        </Button>
      </Box>
    </Box>
  );
}
