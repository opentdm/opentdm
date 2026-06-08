import { Fragment, ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  GearIcon,
  KeyIcon,
  PaintbrushIcon,
  PeopleIcon,
  PulseIcon,
  RepoIcon,
  SearchIcon,
} from "@primer/octicons-react";
import { useProjectsCtx } from "../lib/projects";

interface Cmd {
  group: string;
  label: string;
  meta?: string;
  icon: ReactNode;
  to: string;
}

// ⌘K palette. Indexes projects (from context) and static nav pages — no extra
// fetches. Cross-project object search is intentionally out of scope (would need
// per-project fetches or a backend search endpoint).
export default function CommandPalette({ open, onClose }: { open: boolean; onClose: () => void }) {
  const nav = useNavigate();
  const { projects } = useProjectsCtx();
  const [q, setQ] = useState("");
  const [sel, setSel] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    setQ("");
    setSel(0);
    const t = setTimeout(() => inputRef.current?.focus(), 20);
    return () => clearTimeout(t);
  }, [open]);

  const all = useMemo<Cmd[]>(() => {
    const projectCmds: Cmd[] = projects.map((p) => ({
      group: "Projects",
      label: p.name,
      meta: p.slug,
      icon: <RepoIcon />,
      to: `/projects/${p.slug}`,
    }));
    const pages: Cmd[] = [
      { group: "Go to", label: "All projects", icon: <RepoIcon />, to: "/" },
      { group: "Go to", label: "Settings", icon: <GearIcon />, to: "/settings/account" },
      { group: "Go to", label: "Access tokens", icon: <KeyIcon />, to: "/settings/tokens" },
      { group: "Go to", label: "Appearance", icon: <PaintbrushIcon />, to: "/settings/appearance" },
      { group: "Go to", label: "Activity", icon: <PulseIcon />, to: "/settings/activity" },
      { group: "Go to", label: "Users", icon: <PeopleIcon />, to: "/settings/users" },
    ];
    return [...projectCmds, ...pages];
  }, [projects]);

  const results = useMemo(() => {
    const ql = q.trim().toLowerCase();
    return ql ? all.filter((c) => `${c.label} ${c.meta ?? ""}`.toLowerCase().includes(ql)) : all;
  }, [all, q]);

  // Keep the selection in range as the result set shrinks/grows.
  useEffect(() => {
    setSel((s) => Math.min(s, Math.max(0, results.length - 1)));
  }, [results.length]);

  function go(c: Cmd) {
    nav(c.to);
    onClose();
  }

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSel((s) => Math.min(s + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSel((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const r = results[sel];
        if (r) go(r);
      } else if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, results, sel]);

  if (!open) return null;

  let lastGroup: string | null = null;
  return (
    <div className="otdm-cmdk-scrim" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="otdm-cmdk" role="dialog" aria-modal="true" aria-label="Command palette">
        <div className="otdm-cmdk-input">
          <SearchIcon size={16} />
          <input
            ref={inputRef}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setSel(0);
            }}
            placeholder="Jump to a project or page…"
            aria-label="Search projects and pages"
          />
          <kbd>esc</kbd>
        </div>
        <div className="otdm-cmdk-list">
          {results.length === 0 && <div className="otdm-cmdk-empty">No matches.</div>}
          {results.map((r, i) => {
            const head = r.group !== lastGroup ? (lastGroup = r.group) : null;
            return (
              <Fragment key={`${r.group}:${r.to}`}>
                {head && <div className="otdm-cmdk-group">{head}</div>}
                <button
                  type="button"
                  className={`otdm-cmdk-opt ${i === sel ? "active" : ""}`}
                  onMouseEnter={() => setSel(i)}
                  onClick={() => go(r)}
                >
                  <span className="otdm-cmdk-ico">{r.icon}</span>
                  <span className="otdm-cmdk-label">{r.label}</span>
                  {r.meta && <span className="otdm-cmdk-meta">{r.meta}</span>}
                </button>
              </Fragment>
            );
          })}
        </div>
      </div>
    </div>
  );
}
