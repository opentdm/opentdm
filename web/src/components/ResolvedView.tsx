import { useEffect, useState } from "react";
import { Box, Button, Checkbox, Flash, FormControl, Select } from "../ui/primer";
import { CopyIcon } from "@primer/octicons-react";
import { api, Config, Environment } from "../api";

const FORMATS = ["dotenv", "json", "shell", "yaml", "properties"];

interface ResolvedViewProps {
  slug: string;
  config: Config;
  envs: Environment[];
  initialEnv?: string;
}

// ResolvedView renders a single config's resolved env (base → env override) for a
// chosen environment + format, hitting the per-file resolve endpoint. Secrets are
// masked by default. Used both inside the object page and in the "View resolved"
// modal — it auto-resolves on mount and whenever the selectors change.
export default function ResolvedView({ slug, config, envs, initialEnv }: ResolvedViewProps) {
  const defaultEnv = initialEnv || envs.find((e) => e.is_default)?.slug || envs[0]?.slug || "";
  const [env, setEnv] = useState(defaultEnv);
  const [format, setFormat] = useState("dotenv");
  const [showSecrets, setShowSecrets] = useState(false);
  const [out, setOut] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    if (!env) return;
    let cancelled = false;
    setErr("");
    setCopied(false);
    setLoading(true);
    api
      .resolveConfigText(slug, config.id, env, format, showSecrets)
      .then((t) => {
        if (!cancelled) setOut(t);
      })
      .catch((e: any) => {
        if (!cancelled) {
          setErr(e.message);
          setOut("");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [slug, config.id, env, format, showSecrets, nonce]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(out);
      setCopied(true);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <Box>
      <Box sx={{ display: "flex", gap: 2, alignItems: "flex-end", flexWrap: "wrap", mb: 2 }}>
        <FormControl>
          <FormControl.Label>Environment</FormControl.Label>
          <Select value={env} onChange={(e) => setEnv(e.target.value)}>
            {envs.map((e) => (
              <Select.Option key={e.id} value={e.slug}>
                {e.slug}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <FormControl.Label>Format</FormControl.Label>
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            {FORMATS.map((f) => (
              <Select.Option key={f} value={f}>
                {f}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <Checkbox checked={showSecrets} onChange={(e) => setShowSecrets(e.target.checked)} />
          <FormControl.Label>Show secrets</FormControl.Label>
        </FormControl>
        <Button variant="primary" onClick={() => setNonce((n) => n + 1)} disabled={loading || !env}>
          {loading ? "Resolving…" : "Resolve"}
        </Button>
      </Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      {out && (
        <Box>
          <Box sx={{ display: "flex", justifyContent: "flex-end", mb: 1 }}>
            <Button size="small" leadingVisual={CopyIcon} onClick={copy}>
              {copied ? "Copied" : "Copy"}
            </Button>
          </Box>
          <Box
            as="pre"
            sx={{
              p: 3,
              m: 0,
              bg: "canvas.subtle",
              borderRadius: 2,
              fontFamily: "mono",
              fontSize: 1,
              overflow: "auto",
              whiteSpace: "pre-wrap",
            }}
          >
            {out}
          </Box>
        </Box>
      )}
    </Box>
  );
}
