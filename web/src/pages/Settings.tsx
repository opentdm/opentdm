import { FormEvent, useEffect, useState } from "react";
import { Box, Button, Flash, FormControl, Heading, Label, Text, TextInput } from "../ui/primer";
import { api, PAT } from "../api";

export default function Settings() {
  const [pats, setPats] = useState<PAT[]>([]);
  const [name, setName] = useState("");
  const [days, setDays] = useState("90");
  const [minted, setMinted] = useState("");
  const [err, setErr] = useState("");

  async function load() {
    try {
      setPats(await api.get<PAT[]>("/pats"));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  async function create(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setMinted("");
    try {
      const res = await api.post<{ token: string }>("/pats", { name, expires_in_days: Number(days) || 0 });
      setMinted(res.token);
      setName("");
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function revoke(id: string) {
    try {
      await api.del(`/pats/${id}`);
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Personal access tokens</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Use a PAT (<code>otdmu_…</code>) with the CLI to write config: <code>opentdm login --token otdmu_…</code>.
        A PAT grants your full account access — keep it secret.
      </Text>
      {minted && (
        <Flash variant="warning" sx={{ mb: 3 }}>
          Copy your token now — it won't be shown again:
          <Box as="code" sx={{ display: "block", mt: 1, fontFamily: "mono", wordBreak: "break-all" }}>
            {minted}
          </Box>
        </Flash>
      )}
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box as="form" onSubmit={create} sx={{ display: "flex", gap: 2, alignItems: "flex-end", mb: 3, flexWrap: "wrap" }}>
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="laptop-cli" />
        </FormControl>
        <FormControl>
          <FormControl.Label>Expires (days, 0 = never)</FormControl.Label>
          <TextInput value={days} onChange={(e) => setDays(e.target.value)} />
        </FormControl>
        <Button type="submit" variant="primary">
          Generate token
        </Button>
      </Box>
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {pats.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No tokens.</Box>}
        {pats.map((p) => (
          <Box
            key={p.id}
            sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted", display: "flex", gap: 2, alignItems: "center" }}
          >
            <Text sx={{ fontWeight: "bold" }}>{p.name}</Text>
            <Text sx={{ fontFamily: "mono", color: "fg.muted" }}>{p.prefix}…</Text>
            {p.revoked_at ? <Label variant="danger">revoked</Label> : <Label variant="success">active</Label>}
            <Box sx={{ flex: 1 }} />
            {!p.revoked_at && (
              <Button variant="danger" size="small" onClick={() => revoke(p.id)}>
                Revoke
              </Button>
            )}
          </Box>
        ))}
      </Box>
    </Box>
  );
}
