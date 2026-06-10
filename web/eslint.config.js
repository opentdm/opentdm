import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import globals from "globals";

// Flat config (ESLint 9). The headline rule is no-restricted-imports banning
// @primer/react: every component must import Primer through src/ui/primer (the
// sx shim) so `sx` keeps working — the one exception is the shim itself.
export default tseslint.config(
  { ignores: ["node_modules", "dist", "e2e/"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "no-restricted-imports": [
        "error",
        {
          paths: [
            {
              name: "@primer/react",
              message: "Import Primer from src/ui/primer (the sx shim), never @primer/react directly.",
            },
          ],
          patterns: [
            {
              group: ["@primer/react/*"],
              message: "Import Primer from src/ui/primer (the sx shim), never @primer/react directly.",
            },
          ],
        },
      ],
    },
  },
  {
    // The shim is the single legitimate place to import @primer/react, and it
    // intentionally re-exports Primer plus helper values (not only components),
    // so the react-refresh component-only check doesn't apply to it.
    files: ["src/ui/primer.tsx"],
    rules: {
      "no-restricted-imports": "off",
      "react-refresh/only-export-components": "off",
    },
  },
);
