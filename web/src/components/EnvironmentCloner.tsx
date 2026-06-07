import { useEffect, useState } from "react";
import { Box, Button, Checkbox, ConfirmationDialog, Flash, FormControl, Heading, Select, Text } from "../ui/primer";
import { api, CloneSummary, Environment } from "../api";

interface EnvironmentClonerProps {
  slug: string;
}

// Bulk clone: copy every object's content from one environment layer into
// another in one shot. Each object is cloned + versioned independently (not
// globally atomic); the server returns a per-object summary.
export default function EnvironmentCloner({ slug }: EnvironmentClonerProps) {
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [withValues, setWithValues] = useState(true);
  const [confirm, setConfirm] = useState(false);
  const [summary, setSummary] = useState<CloneSummary | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api
      .listEnvs(slug)
      .then((e) => {
        setEnvs(e);
        const def = e.find((x) => x.is_default)?.slug ?? e[0]?.slug ?? "";
        setFrom(def);
        setTo(e.find((x) => x.slug !== def)?.slug ?? "");
      })
      .catch((e: any) => setErr(e.message));
  }, [slug]);

  const layers = ["base", ...envs.map((e) => e.slug)];

  async function apply() {
    setConfirm(false);
    setErr("");
    setSummary(null);
    try {
      setSummary(await api.cloneEnvironment(slug, { from, to, with_values: withValues }));
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Clone environment</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Copy every object's content from one environment into another.
      </Text>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      <Box sx={{ display: "flex", gap: 3, alignItems: "flex-end", flexWrap: "wrap" }}>
        <FormControl>
          <FormControl.Label>From</FormControl.Label>
          <Select value={from} onChange={(e) => setFrom(e.target.value)}>
            {layers.map((l) => (
              <Select.Option key={l} value={l}>
                {l}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <FormControl.Label>To</FormControl.Label>
          <Select value={to} onChange={(e) => setTo(e.target.value)}>
            {layers.map((l) => (
              <Select.Option key={l} value={l}>
                {l}
              </Select.Option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <Checkbox checked={withValues} onChange={(e) => setWithValues(e.target.checked)} />
          <FormControl.Label>Copy values</FormControl.Label>
        </FormControl>
        <Button variant="primary" disabled={!from || !to || from === to} onClick={() => setConfirm(true)}>
          Clone
        </Button>
      </Box>

      {summary && <CloneResult summary={summary} />}

      {confirm && (
        <ConfirmationDialog
          title={`Clone ${from} → ${to}?`}
          confirmButtonContent="Clone all objects"
          confirmButtonType="danger"
          onClose={(gesture) => (gesture === "confirm" ? void apply() : setConfirm(false))}
        >
          This replaces the <b>{to}</b> layer of every object with content from <b>{from}</b>
          {!withValues ? " (keys only, with empty values that hide inherited base values)" : ""}. Each object can be
          rolled back from its History.
        </ConfirmationDialog>
      )}
    </Box>
  );
}

function CloneResult({ summary }: { summary: CloneSummary }) {
  const line = (label: string, names: string[]) =>
    names.length > 0 && (
      <Text sx={{ display: "block", fontSize: 1 }}>
        <b>{label}:</b> {names.join(", ")}
      </Text>
    );
  return (
    <Box sx={{ mt: 3, p: 3, bg: "canvas.subtle", borderRadius: 2 }}>
      <Text sx={{ fontWeight: "bold", display: "block", mb: 1 }}>
        Cloned {summary.cloned.length}, unchanged {summary.unchanged.length}, skipped {summary.skipped.length}, failed{" "}
        {summary.failed.length}
      </Text>
      {line("Cloned", summary.cloned)}
      {line("Unchanged", summary.unchanged)}
      {line("Skipped", summary.skipped)}
      {summary.failed.length > 0 && (
        <Text sx={{ display: "block", color: "danger.fg", fontSize: 1 }}>
          <b>Failed:</b> {summary.failed.map((f) => `${f.config} (${f.reason})`).join(", ")}
        </Text>
      )}
    </Box>
  );
}
