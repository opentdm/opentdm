import { CSSProperties } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Label } from "../ui/primer";
import { RepoIcon, StarIcon, StarFillIcon, FileDirectoryIcon, ServerIcon, PeopleIcon } from "@primer/octicons-react";
import { Project } from "../api";
import { hueFromString } from "../lib/color";

const ROLE_VARIANT: Record<string, "done" | "accent" | "secondary"> = {
  owner: "done",
  editor: "accent",
  viewer: "secondary",
};

interface ProjectCardProps {
  project: Project;
  isFav: boolean;
  onToggleFav: (slug: string) => void;
}

export default function ProjectCard({ project, isFav, onToggleFav }: ProjectCardProps) {
  const role = project.your_role ?? "viewer";

  return (
    <div className="otdm-pcard-wrap">
      <RouterLink to={`/projects/${project.slug}`} className="otdm-pcard">
        <div className="otdm-pcard-top">
          <span
            className="otdm-proj-ico"
            style={{ width: 38, height: 38, "--h": hueFromString(project.slug) } as CSSProperties}
          >
            <RepoIcon size={20} />
          </span>
          <div className="otdm-pcard-id">
            <div className="otdm-pcard-name">{project.name}</div>
            <div className="otdm-pcard-slug">{project.slug}</div>
          </div>
          <Label variant={ROLE_VARIANT[role] ?? "secondary"}>{role}</Label>
        </div>
        <div className="otdm-pcard-desc">{project.description || "No description."}</div>
        <div className="otdm-pcard-foot">
          <span className="cell"><FileDirectoryIcon size={14} />{project.object_count ?? 0} objects</span>
          <span>·</span>
          <span className="cell"><ServerIcon size={14} />{project.env_count ?? 0} envs</span>
          <span>·</span>
          <span className="cell"><PeopleIcon size={14} />{project.member_count ?? 0} members</span>
        </div>
      </RouterLink>
      <button
        type="button"
        className={`otdm-star ${isFav ? "on" : ""}`}
        aria-label={isFav ? `Unpin ${project.name}` : `Pin ${project.name}`}
        aria-pressed={isFav}
        title={isFav ? "Unpin" : "Pin to top"}
        onClick={() => onToggleFav(project.slug)}
      >
        {isFav ? <StarFillIcon size={16} /> : <StarIcon size={16} />}
      </button>
    </div>
  );
}
