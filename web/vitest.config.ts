import { defineConfig } from "vitest/config";

// The lib/ tests are pure (no DOM): default node environment is enough.
export default defineConfig({
  test: {
    include: ["src/**/*.test.ts"],
  },
});
