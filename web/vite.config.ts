import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The production build is emitted into the Go server module so it can be
// embedded into the binary via go:embed. In dev, /api is proxied to the server.
export default defineConfig({
  plugins: [react()],
  base: "/",
  build: {
    outDir: "../server/internal/webui/dist",
    emptyOutDir: true,
    chunkSizeWarningLimit: 1200,
    // Stable, hash-free filenames: the build is embedded into (and versioned
    // with) the Go binary, so content-hash cache-busting is unnecessary and
    // would churn the committed embed on every build.
    rollupOptions: {
      output: {
        entryFileNames: "assets/app.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
