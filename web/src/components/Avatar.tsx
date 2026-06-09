import { CSSProperties } from "react";
import { hueFromString } from "../lib/color";

// Initials for an avatar: first letters of the first two name parts, else the
// first two characters (matches the Profile pane's initials() helper).
function initials(name: string): string {
  const parts = name.split(/[.\-_@\s]+/).filter(Boolean);
  const letters = parts.length >= 2 ? parts[0][0] + parts[1][0] : name.slice(0, 2);
  return letters.toUpperCase();
}

interface AvatarProps {
  name: string;
  size?: number;
}

// A circular initials avatar, tinted by a stable per-name OKLCH hue (the API has
// no avatar field). See .otdm-avatar in primitives.css.
export default function Avatar({ name, size = 28 }: AvatarProps) {
  return (
    <span
      className="otdm-avatar"
      style={{ width: size, height: size, fontSize: Math.round(size * 0.4), "--h": hueFromString(name) } as CSSProperties}
      aria-hidden="true"
    >
      {initials(name)}
    </span>
  );
}
