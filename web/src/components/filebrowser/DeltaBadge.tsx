import { useEffect, useState } from "react";
import { Label } from "../../ui/primer";
import { api, Config } from "../../api";
import { buildRows, envDelta } from "../../lib/resolve";

interface DeltaBadgeProps {
  slug: string;
  config: Config;
  env: string;
  refreshToken?: number;
}

// How an environment diverges from base for a variable object: "N vs base" (override
// + new + unset rows) or "same as base". Nothing for the base layer or file objects.
export default function DeltaBadge({ slug, config, env, refreshToken }: DeltaBadgeProps) {
  const [n, setN] = useState<number | null>(null);

  useEffect(() => {
    if (config.kind !== "variable" || env === "base") {
      setN(null);
      return;
    }
    let cancelled = false;
    Promise.all([api.getItems(slug, config.id, "base"), api.getItems(slug, config.id, env)])
      .then(([b, l]) => {
        if (cancelled) return;
        const d = envDelta(buildRows(b, l, false));
        setN(d.override + d.new + d.unset);
      })
      .catch(() => !cancelled && setN(null));
    return () => {
      cancelled = true;
    };
  }, [slug, config.id, config.kind, env, refreshToken]);

  if (n === null) return null;
  return n > 0 ? <Label variant="attention">{n} vs base</Label> : <Label variant="secondary">same as base</Label>;
}
