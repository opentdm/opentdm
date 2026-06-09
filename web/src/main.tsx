import React from "react";
import { createRoot } from "react-dom/client";
import { BaseStyles, Box } from "./ui/primer";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import { ColorModeProvider } from "./lib/colorMode";
import { ToastProvider } from "./lib/toast";
// Self-hosted UI + mono fonts (must load before the styles that reference them).
import "./ui/fonts";
// Primer 38 functional color variables, mode-aware (light/dark). The legacy
// ThemeProvider/BaseStyles path doesn't define these, but Primer's component CSS
// and the sx compat shim reference them — without these imports they're undefined
// and dark mode falls back to light. Selectors match colorMode="auto".
import "@primer/primitives/dist/css/functional/themes/light.css";
import "@primer/primitives/dist/css/functional/themes/dark.css";
// tokens.css overrides Primer's accent + font-stack vars; must come AFTER the
// Primer theme CSS so its equal-specificity rules win on cascade order.
import "./ui/tokens.css";
import "./ui/compat.css";
import "./ui/anim.css";
import "./ui/primitives.css";
import "./ui/topbar.css";
import "./ui/shell.css";
import "./ui/settings.css";
import "./ui/cmdk.css";
import "./ui/dropzone.css";
import "./ui/toast.css";
import "./ui/filebrowser.css";

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ColorModeProvider>
      <BaseStyles>
        {/* Fill the viewport with the themed canvas so the background follows
            light/dark mode (otherwise dark-mode text renders on a white page). */}
        <Box sx={{ minHeight: "100vh", bg: "canvas.default", color: "fg.default" }}>
          <ToastProvider>
            <BrowserRouter>
              <App />
            </BrowserRouter>
          </ToastProvider>
        </Box>
      </BaseStyles>
    </ColorModeProvider>
  </React.StrictMode>,
);
