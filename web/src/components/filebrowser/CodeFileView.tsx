import { ReactNode, useEffect, useState } from "react";
import { Box, Flash, Spinner } from "../../ui/primer";
import { api, Config } from "../../api";
import { errMessage } from "../../lib/errors";
import { buildRows, resolvedEntries, ResolvedEntry } from "../../lib/resolve";

const MASK = "••••••••";

interface CodeFileViewProps {
  slug: string;
  config: Config;
  env: string; // "base" or an environment slug
  reveal: boolean;
  refreshToken?: number;
  onText?: (text: string) => void; // pass a stable setter (the displayed text, for Copy)
}

// CodeFileView is the read-only, line-numbered "single view": resolved KEY=value for
// variable objects (syntax-tinted, secrets masked unless reveal), or the per-env file
// content for file objects (the GET falls back to base server-side).
export default function CodeFileView({ slug, config, env, reveal, refreshToken, onText }: CodeFileViewProps) {
  const isVar = config.kind === "variable";
  const [entries, setEntries] = useState<ResolvedEntry[]>([]);
  const [fileLines, setFileLines] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  useEffect(() => {
    let cancelled = false;
    setErr("");
    setLoading(true);
    const load = async () => {
      if (isVar) {
        const isBase = env === "base";
        const [baseItems, layerItems] = isBase
          ? [await api.getItems(slug, config.id, "base"), []]
          : await Promise.all([api.getItems(slug, config.id, "base"), api.getItems(slug, config.id, env)]);
        if (!cancelled) setEntries(resolvedEntries(buildRows(baseItems, layerItems, isBase)));
      } else {
        const text = await api.getText(`/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(env)}`);
        if (!cancelled) setFileLines(text.split("\n"));
      }
    };
    load()
      .catch((e) => !cancelled && setErr(errMessage(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [slug, config.id, isVar, env, refreshToken]);

  const varLines = entries.map((e) => `${e.key}=${e.is_secret && !reveal ? MASK : e.value}`);
  const display = isVar ? varLines : fileLines;
  const text = display.join("\n");

  // Report the displayed text up for the toolbar Copy button (string dep → no loop).
  useEffect(() => {
    onText?.(text);
  }, [text, onText]);

  if (loading)
    return (
      <Box sx={{ p: 3 }}>
        <Spinner size="small" />
      </Box>
    );
  if (err)
    return (
      <Flash variant="danger" sx={{ m: 2 }}>
        {err}
      </Flash>
    );
  if (isVar && entries.length === 0) return <div className="otdm-cf-empty"># (empty for this environment)</div>;

  return (
    <div className="otdm-cf">
      {display.map((ln, i) => (
        <div className="ln-row" key={i}>
          <span className="ln">{i + 1}</span>
          <span className="lc">{isVar ? tintLine(ln) : ln}</span>
        </div>
      ))}
    </div>
  );
}

// Syntax tint for a resolved KEY=value line: key in accent, secret values (masked
// with •) in red, numeric values in amber. Mirrors the prototype's tintCode.
function tintLine(line: string): ReactNode {
  const eq = line.indexOf("=");
  if (eq < 0) return line;
  const k = line.slice(0, eq);
  const v = line.slice(eq + 1);
  const cls = v.includes("•") ? "c-sec" : /^-?\d/.test(v) ? "c-num" : "";
  return (
    <>
      <span className="c-key">{k}</span>={cls ? <span className={cls}>{v}</span> : v}
    </>
  );
}
