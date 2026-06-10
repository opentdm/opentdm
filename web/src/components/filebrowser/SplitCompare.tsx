import { useEffect, useState } from "react";
import { Box, Button, Flash, IconButton, Spinner, Text } from "../../ui/primer";
import { PlusIcon, XIcon } from "@primer/octicons-react";
import { api, Config, Environment } from "../../api";
import { errMessage } from "../../lib/errors";
import { buildResolvedMap, buildRows, diffLines, diffMaps, ResolvedEntry } from "../../lib/resolve";
import BranchEnvMenu from "./BranchEnvMenu";

const MASK = "••••••••";

export interface Pane {
  env: string;
}

interface SplitCompareProps {
  slug: string;
  config: Config;
  envs: Environment[];
  panes: Pane[];
  setPanes: (updater: (p: Pane[]) => Pane[]) => void;
  reveal: boolean;
  refreshToken?: number;
}

// SplitCompare shows 2–3 panes side by side, each at its own environment (its own
// branch dropdown), highlighting where they differ: amber when a key/line differs
// across panes, red when it is absent from a pane. Variables compare by the union
// of resolved keys; files compare line-by-line.
export default function SplitCompare({ slug, config, envs, panes, setPanes, reveal, refreshToken }: SplitCompareProps) {
  const isVar = config.kind === "variable";
  const envList = ["base", ...envs.map((e) => e.slug)];
  const [maps, setMaps] = useState<Map<string, ResolvedEntry>[]>([]);
  const [cols, setCols] = useState<string[][]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  const paneKey = panes.map((p) => p.env).join("|");
  useEffect(() => {
    let cancelled = false;
    setErr("");
    setLoading(true);
    const load = async () => {
      if (isVar) {
        const baseItems = await api.getItems(slug, config.id, "base");
        const ms = await Promise.all(
          panes.map(async (p) => {
            if (p.env === "base") return buildResolvedMap(buildRows(baseItems, [], true));
            const layerItems = await api.getItems(slug, config.id, p.env);
            return buildResolvedMap(buildRows(baseItems, layerItems, false));
          }),
        );
        if (!cancelled) setMaps(ms);
      } else {
        const cs = await Promise.all(
          panes.map((p) =>
            api
              .getText(`/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(p.env)}`)
              .then((t) => t.split("\n")),
          ),
        );
        if (!cancelled) setCols(cs);
      }
    };
    load()
      .catch((e) => !cancelled && setErr(errMessage(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [slug, config.id, isVar, paneKey, refreshToken]);

  const setEnvAt = (i: number, e: string) => setPanes((ps) => ps.map((p, x) => (x === i ? { env: e } : p)));
  const removePane = (i: number) => setPanes((ps) => ps.filter((_, x) => x !== i));
  const addPane = () => setPanes((ps) => [...ps, { env: envList.find((e) => !ps.some((p) => p.env === e)) || "base" }]);

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

  const varDiff = isVar ? diffMaps(maps) : { keys: [] as string[], diff: new Set<string>() };
  const fileDiff = !isVar ? diffLines(cols) : { max: 0, diff: new Set<number>() };
  const diffCount = isVar ? varDiff.diff.size : fileDiff.diff.size;
  const total = isVar ? varDiff.keys.length : fileDiff.max;

  return (
    <Box>
      <div className="otdm-cmp-legend">
        <span>
          <span className="sw otdm-sw-diff" />
          differs
        </span>
        <span>
          <span className="sw otdm-sw-absent" />
          absent
        </span>
        {panes.length < 3 && (
          <Button size="small" leadingVisual={PlusIcon} onClick={addPane}>
            Add pane
          </Button>
        )}
        <Box sx={{ flex: 1 }} />
        <Text sx={{ color: "fg.muted", fontSize: 0 }}>
          {diffCount} of {total} {isVar ? "keys" : "lines"} differ
        </Text>
      </div>
      <div className="otdm-cmp-grid" style={{ gridTemplateColumns: `repeat(${panes.length}, minmax(240px, 1fr))` }}>
        {panes.map((p, ci) => (
          <div className="otdm-cmp-col" key={ci}>
            <div className="otdm-cmp-col-hd">
              <BranchEnvMenu value={p.env} envs={envs} onChange={(e) => setEnvAt(ci, e)} />
              {panes.length > 2 && (
                <Box sx={{ ml: "auto" }}>
                  <IconButton
                    icon={XIcon}
                    aria-label="remove pane"
                    size="small"
                    variant="invisible"
                    onClick={() => removePane(ci)}
                  />
                </Box>
              )}
            </div>
            {isVar
              ? varDiff.keys.map((k) => {
                  const e = maps[ci]?.get(k);
                  const cls = !e ? "absent" : varDiff.diff.has(k) ? "diff" : "";
                  const v = e ? (e.is_secret && !reveal ? MASK : e.value) : null;
                  return (
                    <div className={`otdm-cmp-row ${cls}`} key={k}>
                      <span className="k">{k}</span>
                      {v !== null ? `=${v}` : "  — absent —"}
                    </div>
                  );
                })
              : Array.from({ length: fileDiff.max }).map((_, i) => {
                  const ln = cols[ci]?.[i];
                  const cls = ln == null ? "absent" : fileDiff.diff.has(i) ? "diff" : "";
                  return (
                    <div className={`otdm-cmp-row ${cls}`} key={i}>
                      {ln == null ? " " : ln || " "}
                    </div>
                  );
                })}
          </div>
        ))}
      </div>
    </Box>
  );
}
