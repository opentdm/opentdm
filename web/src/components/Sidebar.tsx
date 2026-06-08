import { CSSProperties } from "react";
import { Link as RouterLink, useLocation } from "react-router-dom";
import { IconButton } from "../ui/primer";
import {
  PackageIcon,
  RepoIcon,
  StarIcon,
  StarFillIcon,
  GearIcon,
  PeopleIcon,
  PulseIcon,
  SignOutIcon,
  PersonIcon,
} from "@primer/octicons-react";
import { User } from "../api";
import { useProjectsCtx } from "../lib/projects";
import { useFavourites } from "../lib/favourites";
import { hueFromString } from "../lib/color";

export default function Sidebar({ me, onSignOut }: { me: User; onSignOut: () => void }) {
  const { pathname } = useLocation();
  const { projects } = useProjectsCtx();
  const { favs } = useFavourites();

  const pinned = projects.filter((p) => favs.has(p.slug));
  const inProject = (slug: string) => pathname === `/projects/${slug}` || pathname.startsWith(`/projects/${slug}/`);

  return (
    <aside className="otdm-sidebar">
      <RouterLink to="/" className="otdm-sb-brand">
        <PackageIcon size={22} />
        opentdm
      </RouterLink>

      <div className="otdm-sb-scroll">
        <div className="otdm-sb-section">Projects</div>
        <nav>
          <RouterLink to="/" className={`otdm-sb-item ${pathname === "/" ? "active" : ""}`}>
            <span className="otdm-sb-ico">
              <RepoIcon size={16} />
            </span>
            <span className="otdm-sb-label">All projects</span>
            <span className="otdm-sb-count">{projects.length}</span>
          </RouterLink>
        </nav>

        <div className="otdm-sb-section">
          <StarIcon size={12} />
          Favourites
        </div>
        <nav>
          {pinned.length === 0 ? (
            <div className="otdm-sb-empty">Star a project to pin it here.</div>
          ) : (
            pinned.map((p) => (
              <RouterLink
                key={p.id}
                to={`/projects/${p.slug}`}
                className={`otdm-sb-item ${inProject(p.slug) ? "active" : ""}`}
              >
                <span
                  className="otdm-proj-ico"
                  style={{ width: 16, height: 16, "--h": hueFromString(p.slug) } as CSSProperties}
                >
                  <RepoIcon size={10} />
                </span>
                <span className="otdm-sb-label">{p.name}</span>
                <span className="otdm-sb-ico">
                  <StarFillIcon size={12} />
                </span>
              </RouterLink>
            ))
          )}
        </nav>
      </div>

      <div className="otdm-sb-foot">
        <RouterLink to="/settings" className={`otdm-sb-item ${pathname.startsWith("/settings") ? "active" : ""}`}>
          <span className="otdm-sb-ico">
            <GearIcon size={16} />
          </span>
          <span className="otdm-sb-label">Settings</span>
        </RouterLink>
        {me.is_admin && (
          <>
            <RouterLink to="/activity" className={`otdm-sb-item ${pathname === "/activity" ? "active" : ""}`}>
              <span className="otdm-sb-ico">
                <PulseIcon size={16} />
              </span>
              <span className="otdm-sb-label">Activity</span>
            </RouterLink>
            <RouterLink to="/users" className={`otdm-sb-item ${pathname === "/users" ? "active" : ""}`}>
              <span className="otdm-sb-ico">
                <PeopleIcon size={16} />
              </span>
              <span className="otdm-sb-label">Users</span>
            </RouterLink>
          </>
        )}
        <div className="otdm-sb-user">
          <span className="otdm-sb-ico">
            <PersonIcon size={16} />
          </span>
          <span className="otdm-sb-label" title={me.email}>
            {me.username}
          </span>
          <IconButton icon={SignOutIcon} aria-label="Sign out" variant="invisible" size="small" onClick={onSignOut} />
        </div>
      </div>
    </aside>
  );
}
