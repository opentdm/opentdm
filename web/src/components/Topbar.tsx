import { useEffect, useState } from "react";
import { Link as RouterLink, useLocation } from "react-router-dom";
import { Breadcrumbs, IconButton } from "../ui/primer";
import { CommandPaletteIcon, MoonIcon, SunIcon } from "@primer/octicons-react";
import { api } from "../api";
import { useProjectsCtx } from "../lib/projects";
import { useColorMode } from "../lib/colorMode";

interface Crumb {
  label: string;
  to?: string;
}

interface TopbarProps {
  onOpenPalette: () => void;
}

const SETTINGS_LABELS: Record<string, string> = {
  account: "Profile",
  tokens: "Access tokens",
  appearance: "Appearance",
  activity: "Activity",
  users: "Users",
};

// A slim global topbar showing route-derived breadcrumbs. Replaces the
// per-page Breadcrumbs that each route used to render. The project name comes
// from the shared projects context; the object name is fetched on object routes.
export default function Topbar({ onOpenPalette }: TopbarProps) {
  const { pathname } = useLocation();
  const { projects } = useProjectsCtx();
  const { mode, setMode } = useColorMode();
  const dark = mode === "dark";
  const parts = pathname.split("/").filter(Boolean);
  const slug = parts[0] === "projects" ? parts[1] : "";
  const configId = parts[0] === "projects" && parts[2] === "configs" ? parts[3] : "";
  const [configName, setConfigName] = useState("");

  useEffect(() => {
    if (!slug || !configId) {
      setConfigName("");
      return;
    }
    let alive = true;
    api
      .getConfig(slug, configId)
      .then((c) => alive && setConfigName(c.name))
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [slug, configId]);

  const crumbs = buildCrumbs();

  function buildCrumbs(): Crumb[] {
    if (parts.length === 0) return [{ label: "Projects" }];
    if (parts[0] === "settings") {
      const out: Crumb[] = [
        { label: "Projects", to: "/" },
        { label: "Settings", to: "/settings" },
      ];
      const section = parts[1];
      if (section && SETTINGS_LABELS[section]) out.push({ label: SETTINGS_LABELS[section] });
      return out;
    }
    if (parts[0] === "projects" && slug) {
      const projectName = projects.find((p) => p.slug === slug)?.name ?? slug;
      const out: Crumb[] = [
        { label: "Projects", to: "/" },
        { label: projectName, to: `/projects/${slug}` },
      ];
      if (parts[2] === "configs") out.push({ label: configName || "…" });
      else if (parts[2] === "settings") out.push({ label: "Settings" });
      else if (parts[2] === "activity") out.push({ label: "Activity" });
      return out;
    }
    return [{ label: "Projects", to: "/" }];
  }

  return (
    <div className="otdm-topbar">
      <Breadcrumbs>
        {crumbs.map((c, i) => {
          const isLast = i === crumbs.length - 1;
          return c.to && !isLast ? (
            <Breadcrumbs.Item key={i} as={RouterLink} to={c.to}>
              {c.label}
            </Breadcrumbs.Item>
          ) : (
            <Breadcrumbs.Item key={i} selected>
              {c.label}
            </Breadcrumbs.Item>
          );
        })}
      </Breadcrumbs>
      <div className="otdm-topbar-actions">
        <IconButton
          icon={dark ? SunIcon : MoonIcon}
          aria-label={dark ? "Switch to light theme" : "Switch to dark theme"}
          size="small"
          onClick={() => setMode(dark ? "light" : "dark")}
        />
        <IconButton icon={CommandPaletteIcon} aria-label="Open command palette" size="small" onClick={onOpenPalette} />
      </div>
    </div>
  );
}
