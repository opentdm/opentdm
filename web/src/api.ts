// Typed fetch wrapper for the opentdm API. Sends the session cookie and the
// double-submit CSRF token on mutations.

export interface UserPreferences {
  color_mode?: string;
  favourites?: string[];
}
export interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
  preferences?: UserPreferences;
}
export interface Project {
  id: string;
  slug: string;
  name: string;
  description: string;
  your_role?: string; // viewer | editor | owner (caller's role)
  object_count?: number; // grid summary counts (present on the list endpoint)
  env_count?: number;
  member_count?: number;
  created_at: string;
}
export interface Member {
  user_id: string;
  username: string;
  email: string;
  role: string;
}
export interface AdminUser {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  created_at: string;
}
export interface Invitation {
  id: string;
  email: string;
  role: string;
  expires_at: string;
}
export interface InvitationInfo {
  email: string;
  role: string;
  project: string;
  project_slug: string;
}
export interface InvitationResult {
  id: string;
  email: string;
  role: string;
  email_sent: boolean;
  accept_url?: string; // present only when SMTP is unconfigured
}

// A cross-config key collision in a resolved environment: the winning config
// supplied the value, shadowing the losing one.
export interface Collision {
  key: string;
  winning_config: string;
  losing_config: string;
}

// Role helpers (UX gating only — the server is the enforcement authority).
export const canWrite = (role?: string): boolean => role === "editor" || role === "owner";
export const canManage = (role?: string): boolean => role === "owner";

export interface AuditEntry {
  id: string;
  project_id?: string | null;
  actor: string;
  action: string;
  target_type?: string;
  target_id?: string;
  status: number;
  created_at: string;
}

// Friendly labels for audit action codes (falls back to the raw code).
const AUDIT_LABELS: Record<string, string> = {
  "project.created": "created the project",
  "config.created": "created an object",
  "config.updated": "edited an object",
  "config.archived": "deleted an object",
  "config.items.updated": "updated variables",
  "config.file.updated": "updated a file",
  "config.rolled_back": "rolled back an object",
  "config.cloned": "cloned an object",
  "environment.created": "created an environment",
  "environment.updated": "renamed an environment",
  "environment.deleted": "deleted an environment",
  "environment.reordered": "reordered environments",
  "environment.cloned": "cloned an environment",
  "member.added": "added a member",
  "member.updated": "changed a member's role",
  "member.removed": "removed a member",
  "invitation.created": "invited someone",
  "invitation.revoked": "revoked an invitation",
  "token.created": "created a service token",
  "token.revoked": "revoked a service token",
  "user.updated": "updated a user",
};
export const auditLabel = (action: string): string => AUDIT_LABELS[action] ?? action;
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
  key_count?: number; // base-layer key count (present on the configs list endpoint)
}
export interface Item {
  key: string;
  value: string;
  is_secret: boolean;
  deleted: boolean;
}
export interface SearchHit {
  config_id: string;
  name: string;
  kind?: string; // variable | file
  is_secret?: boolean;
  project_slug: string;
  project_name: string;
}
export interface Token {
  id: string;
  name: string;
  prefix: string;
  scope: string;
  environment_ids: string[];
  created_at: string;
  last_used_at?: string | null;
  revoked_at?: string | null;
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

// auditPage is a GET that also surfaces the meta cursor (request() drops meta).
async function auditPage(path: string): Promise<{ entries: AuditEntry[]; next?: string }> {
  const resp = await fetch("/api/v1" + path, { credentials: "include" });
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
  const json = text ? JSON.parse(text) : {};
  return { entries: (json.data ?? []) as AuditEntry[], next: json.meta?.next };
}

export interface VersionMeta {
  version: number;
  is_current: boolean;
  kind: string;
  byte_size: number;
  comment?: string;
  added?: number; // per-version deltas vs the previous version (variable configs)
  changed?: number;
  removed?: number;
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

  getProject: (slug: string) => request<Project>("GET", `/projects/${slug}`),

  // --- members ---
  listMembers: (slug: string) => request<Member[]>("GET", `/projects/${slug}/members`),
  addMember: (slug: string, body: { user: string; role: string }) =>
    request<Member>("POST", `/projects/${slug}/members`, body),
  updateMember: (slug: string, userId: string, role: string) =>
    request<unknown>("PATCH", `/projects/${slug}/members/${userId}`, { role }),
  removeMember: (slug: string, userId: string) => request<unknown>("DELETE", `/projects/${slug}/members/${userId}`),

  // --- invitations ---
  listInvitations: (slug: string) => request<Invitation[]>("GET", `/projects/${slug}/invitations`),
  createInvitation: (slug: string, body: { email: string; role: string }) =>
    request<InvitationResult>("POST", `/projects/${slug}/invitations`, body),
  revokeInvitation: (slug: string, id: string) => request<unknown>("DELETE", `/projects/${slug}/invitations/${id}`),
  getInvitation: (token: string) => request<InvitationInfo>("GET", `/invitations/${encodeURIComponent(token)}`),
  acceptInvitation: (token: string, body: { username: string; password: string }) =>
    request<User>("POST", `/invitations/${encodeURIComponent(token)}/accept`, body),

  // --- audit / activity (keyset-paginated; returns entries + next cursor) ---
  listProjectAudit: (slug: string, before?: string) =>
    auditPage(`/projects/${slug}/audit?limit=50${before ? `&before=${encodeURIComponent(before)}` : ""}`),
  listAudit: (before?: string) => auditPage(`/audit?limit=50${before ? `&before=${encodeURIComponent(before)}` : ""}`),

  // --- admin user directory ---
  listUsers: () => request<AdminUser[]>("GET", `/users`),
  updateUser: (id: string, body: { is_active?: boolean; is_admin?: boolean }) =>
    request<AdminUser>("PATCH", `/users/${id}`, body),

  // Cross-project object search (⌘K palette).
  searchConfigs: (q: string) => request<SearchHit[]>("GET", `/search?q=${encodeURIComponent(q)}`),

  getConfig: (slug: string, id: string) => request<Config>("GET", `/projects/${slug}/configs/${id}`),
  updateConfig: (slug: string, id: string, body: { name: string; sort_order: number; description: string }) =>
    request<Config>("PATCH", `/projects/${slug}/configs/${id}`, body),
  archiveConfig: (slug: string, id: string) => request<unknown>("DELETE", `/projects/${slug}/configs/${id}`),

  getItems: (slug: string, configId: string, env: string) =>
    request<Item[]>("GET", `/projects/${slug}/configs/${configId}/items?env=${encodeURIComponent(env)}`),
  putItems: (slug: string, configId: string, env: string, items: Item[], comment?: string) =>
    request<unknown>("PUT", `/projects/${slug}/configs/${configId}/items?env=${encodeURIComponent(env)}`, {
      items,
      comment,
    }),
  // Per-file resolve: a single config's effective env (base → env override,
  // tombstones), rendered server-side. Secrets are masked by default so the
  // preview is safe to screen-share.
  resolveConfigText: async (
    slug: string,
    configId: string,
    env: string,
    format: string,
    includeSecrets = false,
  ): Promise<string> => {
    const params = new URLSearchParams({ env, format });
    if (!includeSecrets) params.set("include_secrets", "false");
    const resp = await fetch(
      `/api/v1/projects/${encodeURIComponent(slug)}/configs/${encodeURIComponent(configId)}/resolve?${params.toString()}`,
      { credentials: "include" },
    );
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
    return text;
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
