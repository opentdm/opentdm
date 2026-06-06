import React from "react";
import { createRoot } from "react-dom/client";
import { ThemeProvider, BaseStyles, Box } from "@primer/react";
import { BrowserRouter } from "react-router-dom";
import App from "./App";

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
