import { FormEvent, useEffect, useState } from "react";
import { KeyIcon, PlusIcon } from "@primer/octicons-react";
import { Box, Button, Flash, FormControl, Heading, Label, Text, TextInput } from "../../ui/primer";
import { api, PAT } from "../../api";
import { errMessage } from "../../lib/errors";
import { useToast } from "../../lib/toast";
import Overline from "../Overline";

// Personal access tokens (otdmu_…). Moved verbatim from the old standalone
// Settings page into the consolidated Settings → Account → Access tokens panel.
export default function AccessTokensPanel() {
  const toast = useToast();
  const [pats, setPats] = useState<PAT[]>([]);
  const [name, setName] = useState("");
  const [days, setDays] = useState("90");
  const [minted, setMinted] = useState("");
  const [err, setErr] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  async function load() {
    try {
      setPats(await api.get<PAT[]>("/pats"));
    } catch (e) {
      setErr(errMessage(e));
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
      toast("Token created.");
    } catch (e) {
      setErr(errMessage(e));
    }
  }
  async function revoke(id: string) {
    try {
      await api.del(`/pats/${id}`);
      await load();
      toast("Token revoked.");
    } catch (e) {
      setErr(errMessage(e));
    }
  }

  return (
    <Box>
      <Overline>Account</Overline>
      <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 1 }}>
        <Heading sx={{ fontSize: 3 }}>Personal access tokens</Heading>
        <Box sx={{ flex: 1 }} />
        <Button variant="primary" leadingVisual={PlusIcon} onClick={() => setShowCreate((v) => !v)}>
          New token
        </Button>
      </Box>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        PATs (<code>otdmu_…</code>) act as you for CLI &amp; management writes.
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
      {showCreate && (
        <Box
          as="form"
          onSubmit={create}
          sx={{ display: "flex", gap: 2, alignItems: "flex-end", mb: 3, flexWrap: "wrap" }}
        >
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
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {pats.length === 0 && <Box sx={{ p: 3, color: "fg.muted" }}>No tokens.</Box>}
        {pats.map((p) => (
          <Box
            key={p.id}
            sx={{
              p: 3,
              borderBottomWidth: 1,
              borderBottomStyle: "solid",
              borderColor: "border.muted",
              display: "flex",
              gap: 2,
              alignItems: "center",
            }}
          >
            <Box sx={{ color: "fg.muted", display: "flex" }}>
              <KeyIcon />
            </Box>
            <Text sx={{ fontWeight: "bold" }}>{p.name}</Text>
            <Text sx={{ fontFamily: "mono", color: "fg.muted" }}>{p.prefix}…</Text>
            <Text sx={{ color: "fg.muted" }}>
              {p.expires_at ? `expires ${new Date(p.expires_at).toLocaleDateString()}` : "no expiry"}
            </Text>
            <Text sx={{ color: "fg.muted" }}>
              {p.last_used_at ? `used ${new Date(p.last_used_at).toLocaleDateString()}` : "never used"}
            </Text>
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
