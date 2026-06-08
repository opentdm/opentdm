import { CSSProperties } from "react";
import { Box, Heading, Text } from "../../ui/primer";
import { CheckIcon, DeviceDesktopIcon, MoonIcon, SunIcon } from "@primer/octicons-react";
import { ColorMode, useColorMode } from "../../lib/colorMode";

// Swatch colors are intentionally literal (not functional vars) so each card
// previews its own theme regardless of the currently-active mode.
interface Opt {
  mode: ColorMode;
  label: string;
  icon: typeof SunIcon;
  panel: string;
  bar: string;
}
const OPTIONS: Opt[] = [
  { mode: "light", label: "Light", icon: SunIcon, panel: "#ffffff", bar: "#afb8c1" },
  { mode: "dark", label: "Dark", icon: MoonIcon, panel: "#0d1117", bar: "#3d444d" },
  { mode: "auto", label: "Auto", icon: DeviceDesktopIcon, panel: "linear-gradient(105deg,#ffffff 50%,#0d1117 50%)", bar: "#8b949e" },
];

export default function AppearancePanel() {
  const { mode, setMode } = useColorMode();
  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Appearance</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Choose how opentdm looks. “Auto” follows your system setting.
      </Text>
      <Box className="otdm-theme-cards">
        {OPTIONS.map((o) => {
          const active = mode === o.mode;
          const Icon = o.icon;
          return (
            <button
              key={o.mode}
              type="button"
              className={`otdm-theme-card ${active ? "active" : ""}`}
              aria-pressed={active}
              onClick={() => setMode(o.mode)}
            >
              <span className="otdm-theme-card-hd">
                <Icon size={15} />
                {o.label}
                {active && (
                  <span className="check">
                    <CheckIcon size={15} />
                  </span>
                )}
              </span>
              <span className="otdm-theme-swatch" aria-hidden="true">
                <span className="pane" style={{ background: o.panel } as CSSProperties}>
                  <span className="bar" style={{ background: o.bar, width: "70%" }} />
                  <span className="bar" style={{ background: o.bar, width: "45%" }} />
                </span>
              </span>
            </button>
          );
        })}
      </Box>
    </Box>
  );
}
