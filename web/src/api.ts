// Typed fetch wrapper for the opentdm API. Sends the session cookie and the
// double-submit CSRF token on mutations.

export interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
}
export interface Project {
  id: string;
  slug: string;
  name: string;
  description: string;
  created_at: string;
}
export interface Environment {
  id: string;
  slug: string;
  name: string;
  rank: number;
  is_default: boolean;
}
export interface Config {
  id: string;
  kind: string;
  format: string;
  name: string;
  sort_order: number;
  description: string;
  is_secret: boolean;
  tags: string[];
}
export interface Item {
  key: string;
  value: string;
  is_secret: boolean;
  deleted: boolean;
}
export interface Token {
  id: string;
  name: string;
  prefix: string;
  scope: string;
  environment_ids: string[];
  created_at: string;
}

export interface CloneSummary {
  from: string;
  to: string;
  cloned: string[];
  unchanged: string[];
  skipped: string[];
  failed: { config: string; reason: string }[];
}

export class APIError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

function cookie(name: string): string {
  const m = document.cookie.match(new RegExp("(?:^|; )" + name + "=([^;]*)"));
  return m ? decodeURIComponent(m[1]) : "";
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (method !== "GET") headers["X-CSRF-Token"] = cookie("otdm_csrf");
  const resp = await fetch("/api/v1" + path, {
    method,
    headers,
    credentials: "include",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  const text = await resp.text();
  if (!resp.ok) {
    let msg = resp.statusText;
    try {
      const j = JSON.parse(text);
      msg = j.detail || j.title || msg;
    } catch {
      /* non-JSON */
    }
    throw new APIError(resp.status, msg);
  }
  if (!text) return undefined as T;
  const json = JSON.parse(text);
  return (json.data ?? json) as T;
}

export interface VersionMeta {
  version: number;
  is_current: boolean;
  kind: string;
  byte_size: number;
  comment?: string;
  created_at: string;
}
export interface VarDiffEntry {
  key: string;
  status: string;
  from?: string;
  to?: string;
  was_secret: boolean;
  is_secret: boolean;
}
export interface DiffResult {
  kind: string;
  from: number;
  to: number;
  vars?: VarDiffEntry[];
  file_diff?: string;
}
export interface PAT {
  id: string;
  name: string;
  prefix: string;
  expires_at: string | null;
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
}

export const api = {
  get: <T>(p: string) => request<T>("GET", p),
  post: <T>(p: string, b?: unknown) => request<T>("POST", p, b),
  put: <T>(p: string, b?: unknown) => request<T>("PUT", p, b),
  patch: <T>(p: string, b?: unknown) => request<T>("PATCH", p, b),
  del: <T>(p: string) => request<T>("DELETE", p),

  // --- typed helpers ---
  listEnvs: (slug: string) => request<Environment[]>("GET", `/projects/${slug}/environments`),
  createEnv: (slug: string, name: string) => request<Environment>("POST", `/projects/${slug}/environments`, { name }),
  updateEnv: (slug: string, id: string, body: { slug?: string; name?: string; is_default?: boolean }) =>
    request<Environment>("PATCH", `/projects/${slug}/environments/${id}`, body),
  deleteEnv: (slug: string, id: string) => request<unknown>("DELETE", `/projects/${slug}/environments/${id}`),
  reorderEnvs: (slug: string, orderedIds: string[]) =>
    request<Environment[]>("POST", `/projects/${slug}/environments/reorder`, { ordered_ids: orderedIds }),

  getConfig: (slug: string, id: string) => request<Config>("GET", `/projects/${slug}/configs/${id}`),
  cloneConfigLayer: (slug: string, id: string, body: { from: string; to: string; with_values: boolean }) =>
    request<{ from: string; to: string; version: number }>("POST", `/projects/${slug}/configs/${id}/clone`, body),
  cloneEnvironment: (slug: string, body: { from: string; to: string; with_values: boolean }) =>
    request<CloneSummary>("POST", `/projects/${slug}/clone-environment`, body),
  updateConfig: (slug: string, id: string, body: { name: string; sort_order: number; description: string; tags: string[] }) =>
    request<Config>("PATCH", `/projects/${slug}/configs/${id}`, body),
  archiveConfig: (slug: string, id: string) => request<unknown>("DELETE", `/projects/${slug}/configs/${id}`),

  getItems: (slug: string, configId: string, env: string) =>
    request<Item[]>("GET", `/projects/${slug}/configs/${configId}/items?env=${encodeURIComponent(env)}`),
  putItems: (slug: string, configId: string, env: string, items: Item[], comment?: string) =>
    request<unknown>("PUT", `/projects/${slug}/configs/${configId}/items?env=${encodeURIComponent(env)}`, { items, comment }),
  resolveText: async (project: string, env: string, format: string): Promise<string> => {
    const resp = await fetch(
      `/api/v1/projects/${encodeURIComponent(project)}/resolve?env=${encodeURIComponent(env)}&format=${format}`,
      { credentials: "include" },
    );
    return resp.text();
  },
  // Raw text GET (for file blobs / version snapshots).
  getText: async (path: string): Promise<string> => {
    const resp = await fetch("/api/v1" + path, { credentials: "include" });
    if (!resp.ok) throw new APIError(resp.status, await resp.text());
    return resp.text();
  },
  // Raw body PUT (for file blobs), with CSRF.
  putRaw: async (path: string, body: string, contentType: string): Promise<void> => {
    const resp = await fetch("/api/v1" + path, {
      method: "PUT",
      headers: { "Content-Type": contentType, "X-CSRF-Token": cookie("otdm_csrf") },
      credentials: "include",
      body,
    });
    if (!resp.ok) {
      let msg = resp.statusText;
      try {
        msg = JSON.parse(await resp.text()).detail || msg;
      } catch {
        /* ignore */
      }
      throw new APIError(resp.status, msg);
    }
  },
};
