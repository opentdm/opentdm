import { Link as RouterLink } from "react-router-dom";
import { Label } from "../../ui/primer";
import { FileDirectoryIcon, FileIcon, KeyIcon, LockIcon } from "@primer/octicons-react";
import { Config } from "../../api";

interface FileTreeProps {
  slug: string;
  projectName: string;
  configs: Config[];
  activeId: string;
}

// A variable object reads as <name>.env; a file object keeps its filename.
const objFile = (c: Config) => (c.kind === "variable" ? `${c.name}.env` : c.name);

// FileTree lists a project's objects as a file tree (the project = repo). Each file
// links to its object page, so the tree doubles as sibling navigation.
export default function FileTree({ slug, projectName, configs, activeId }: FileTreeProps) {
  return (
    <div className="otdm-fb-tree">
      <div className="otdm-fb-tree-hd">
        <FileDirectoryIcon size={15} />
        {projectName}
      </div>
      <div className="otdm-fb-tree-body">
        {configs.map((c) => {
          const Icon = c.format === "secret" ? LockIcon : c.kind === "file" ? FileIcon : KeyIcon;
          return (
            <RouterLink
              key={c.id}
              to={`/projects/${slug}/configs/${c.id}`}
              className={`otdm-fb-node${c.id === activeId ? " active" : ""}`}
            >
              <Icon size={13} />
              <span className="nm">{objFile(c)}</span>
              {c.format === "secret" && (
                <Label variant="danger" className="otdm-fb-trail">
                  secret
                </Label>
              )}
            </RouterLink>
          );
        })}
      </div>
    </div>
  );
}
