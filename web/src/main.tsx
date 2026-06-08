import React from "react";
import { createRoot } from "react-dom/client";
import { ThemeProvider, BaseStyles, Box } from "./ui/primer";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
// Primer 38 functional color variables, mode-aware (light/dark). The legacy
// ThemeProvider/BaseStyles path doesn't define these, but Primer's component CSS
// and the sx compat shim reference them — without these imports they're undefined
// and dark mode falls back to light. Selectors match colorMode="auto".
import "@primer/primitives/dist/css/functional/themes/light.css";
import "@primer/primitives/dist/css/functional/themes/dark.css";
import "./ui/compat.css";
import "./ui/shell.css";
import "./ui/filebrowser.css";

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ThemeProvider colorMode="auto">
      <BaseStyles>
        {/* Fill the viewport with the themed canvas so the background follows
            light/dark mode (otherwise dark-mode text renders on a white page). */}
        <Box sx={{ minHeight: "100vh", bg: "canvas.default", color: "fg.default" }}>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </Box>
      </BaseStyles>
    </ThemeProvider>
  </React.StrictMode>,
);
