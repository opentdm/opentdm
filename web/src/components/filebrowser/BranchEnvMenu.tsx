import { ActionList, ActionMenu, Button, Label } from "../../ui/primer";
import { CheckIcon, ChevronDownIcon, GitBranchIcon } from "@primer/octicons-react";
import { Environment } from "../../api";

interface BranchEnvMenuProps {
  value: string; // "base" or an environment slug
  envs: Environment[];
  onChange: (env: string) => void;
}

// Branch-style environment picker (⎇ env: <env> ▾) — the file browser's "branch":
// switching the environment reframes the whole object view as base ⊕ that env.
export default function BranchEnvMenu({ value, envs, onChange }: BranchEnvMenuProps) {
  const options = ["base", ...envs.map((e) => e.slug)];
  return (
    <ActionMenu>
      <ActionMenu.Anchor>
        <Button leadingVisual={GitBranchIcon} trailingVisual={ChevronDownIcon}>
          <span className="otdm-branch-who">env:</span>
          <span className="otdm-branch-val">{value}</span>
        </Button>
      </ActionMenu.Anchor>
      <ActionMenu.Overlay width="small">
        <ActionList>
          <ActionList.Group title="switch environment">
            {options.map((opt) => (
              <ActionList.Item key={opt} onSelect={() => onChange(opt)}>
                <ActionList.LeadingVisual>{opt === value ? <CheckIcon /> : <span />}</ActionList.LeadingVisual>
                <span className="otdm-branch-val">{opt}</span>
                {opt === "base" && (
                  <ActionList.TrailingVisual>
                    <Label variant="secondary">default</Label>
                  </ActionList.TrailingVisual>
                )}
              </ActionList.Item>
            ))}
          </ActionList.Group>
        </ActionList>
      </ActionMenu.Overlay>
    </ActionMenu>
  );
}
