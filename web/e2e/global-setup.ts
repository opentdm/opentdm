import { request } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface SeedData {
  projectSlug: string;
  configId: string;
  envSlugs: string[];
}

async function getCsrfToken(ctx: Awaited<ReturnType<typeof request.newContext>>): Promise<string> {
  const state = await ctx.storageState();
  const csrfCookie = state.cookies.find((c) => c.name === "otdm_csrf");
  return csrfCookie?.value ?? "";
}

function parseData<T>(json: Record<string, unknown>): T {
  return ((json.data ?? json) as T);
}

export default async function globalSetup() {
  const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:18099";
  const setupToken = process.env.E2E_SETUP_TOKEN ?? "";
  const username = process.env.E2E_USERNAME ?? "e2e-admin";
  const email = process.env.E2E_EMAIL ?? "e2e@example.test";
  const password = process.env.E2E_PASSWORD ?? "changeme";

  const authDir = path.join(__dirname, ".auth");
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true });
  }

  const ctx = await request.newContext({ baseURL });

  // Bootstrap or login
  if (setupToken) {
    console.log("[e2e] Bootstrapping admin account…");
    const resp = await ctx.post("/api/v1/auth/bootstrap", {
      data: { setup_token: setupToken, username, email, password },
    });
    if (!resp.ok()) {
      const body = await resp.text();
      throw new Error(`Bootstrap failed (${resp.status()}): ${body}`);
    }
  } else {
    console.log("[e2e] Logging in as existing user…");
    const resp = await ctx.post("/api/v1/auth/login", {
      data: { username, password },
    });
    if (!resp.ok()) {
      const body = await resp.text();
      throw new Error(`Login failed (${resp.status()}): ${body}`);
    }
  }

  // Helper: POST with CSRF
  async function post<T>(path: string, data: unknown): Promise<T> {
    const csrf = await getCsrfToken(ctx);
    const resp = await ctx.post(path, {
      data,
      headers: { "X-CSRF-Token": csrf },
    });
    if (!resp.ok()) {
      const body = await resp.text();
      throw new Error(`POST ${path} failed (${resp.status()}): ${body}`);
    }
    const json = (await resp.json()) as Record<string, unknown>;
    return parseData<T>(json);
  }

  // Helper: PUT with CSRF
  async function put(path: string, data: unknown): Promise<void> {
    const csrf = await getCsrfToken(ctx);
    const resp = await ctx.put(path, {
      data,
      headers: { "X-CSRF-Token": csrf },
    });
    if (!resp.ok()) {
      const body = await resp.text();
      throw new Error(`PUT ${path} failed (${resp.status()}): ${body}`);
    }
  }

  // --- Seed data ---

  // 1. Create project
  console.log("[e2e] Creating project 'Payments API'…");
  const project = await post<{ id: string; slug: string; name: string }>(
    "/api/v1/projects",
    { name: "Payments API" },
  );
  const projectSlug = project.slug;
  console.log(`[e2e] Project slug: ${projectSlug}`);

  // 2. Use the auto-seeded environments (the server creates staging + production for every new project)
  console.log("[e2e] Fetching auto-seeded environments…");
  const envsResp = await ctx.get(`/api/v1/projects/${projectSlug}/environments`);
  if (!envsResp.ok()) {
    throw new Error(`GET environments failed (${envsResp.status()}): ${await envsResp.text()}`);
  }
  const envsJson = (await envsResp.json()) as Record<string, unknown>;
  const envList = (envsJson.data ?? envsJson) as Array<{ id: string; slug: string; name: string }>;
  const envSlugs = envList.map((e) => e.slug);
  console.log(`[e2e] Environments: ${envSlugs.join(", ")}`);

  // Use the first environment slug for staging overrides (typically "staging")
  const stagingEnvSlug = envSlugs[0] ?? "staging";

  // 3. Create variable object
  console.log("[e2e] Creating config 'app-config'…");
  const config = await post<{ id: string; name: string }>(
    `/api/v1/projects/${projectSlug}/configs`,
    { kind: "variable", format: "env", name: "app-config" },
  );
  const configId = config.id;
  console.log(`[e2e] Config id: ${configId}`);

  // 4. Seed base items — version 1: initial set of keys
  console.log("[e2e] Seeding base items v1…");
  await put(`/api/v1/projects/${projectSlug}/configs/${configId}/items?env=base`, {
    items: [
      { key: "APP_ENV", value: "base", is_secret: false, deleted: false },
      { key: "LOG_LEVEL", value: "info", is_secret: false, deleted: false },
      { key: "DB_HOST", value: "localhost", is_secret: false, deleted: false },
      { key: "DB_PORT", value: "5432", is_secret: false, deleted: false },
      { key: "SECRET_KEY", value: "s3cr3t", is_secret: true, deleted: false },
    ],
    comment: "Initial base config",
  });

  // 5. Base items — version 2: change LOG_LEVEL, add MAX_CONNECTIONS
  console.log("[e2e] Seeding base items v2…");
  await put(`/api/v1/projects/${projectSlug}/configs/${configId}/items?env=base`, {
    items: [
      { key: "APP_ENV", value: "base", is_secret: false, deleted: false },
      { key: "LOG_LEVEL", value: "warn", is_secret: false, deleted: false },
      { key: "DB_HOST", value: "localhost", is_secret: false, deleted: false },
      { key: "DB_PORT", value: "5432", is_secret: false, deleted: false },
      { key: "SECRET_KEY", value: "s3cr3t", is_secret: true, deleted: false },
      { key: "MAX_CONNECTIONS", value: "10", is_secret: false, deleted: false },
    ],
    comment: "Bump log level, add connection pool",
  });

  // 6. Base items — version 3: remove DB_PORT (tombstone)
  console.log("[e2e] Seeding base items v3 (remove DB_PORT)…");
  await put(`/api/v1/projects/${projectSlug}/configs/${configId}/items?env=base`, {
    items: [
      { key: "APP_ENV", value: "base", is_secret: false, deleted: false },
      { key: "LOG_LEVEL", value: "warn", is_secret: false, deleted: false },
      { key: "DB_HOST", value: "localhost", is_secret: false, deleted: false },
      { key: "DB_PORT", value: "", is_secret: false, deleted: true },
      { key: "SECRET_KEY", value: "s3cr3t", is_secret: true, deleted: false },
      { key: "MAX_CONNECTIONS", value: "10", is_secret: false, deleted: false },
    ],
    comment: "Remove DB_PORT (use default)",
  });

  // 7. Staging override
  console.log("[e2e] Seeding staging override…");
  await put(
    `/api/v1/projects/${projectSlug}/configs/${configId}/items?env=${stagingEnvSlug}`,
    {
      items: [
        { key: "APP_ENV", value: "staging", is_secret: false, deleted: false },
        { key: "LOG_LEVEL", value: "debug", is_secret: false, deleted: false },
        { key: "DB_HOST", value: "staging-db.internal", is_secret: false, deleted: false },
      ],
      comment: "Staging overrides",
    },
  );

  // Save storage state (cookies)
  const statePath = path.join(authDir, "state.json");
  await ctx.storageState({ path: statePath });
  console.log(`[e2e] Auth state saved to ${statePath}`);

  // Write seed data for specs
  const seedPath = path.join(authDir, "seed.json");
  const seed: SeedData = { projectSlug, configId, envSlugs };
  fs.writeFileSync(seedPath, JSON.stringify(seed, null, 2));
  console.log(`[e2e] Seed data saved to ${seedPath}`);

  await ctx.dispose();
}
