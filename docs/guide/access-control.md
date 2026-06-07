# Access control

opentdm has **per-project roles** plus three authentication planes. Instance admins are implicit owners of
every project.

## Roles

Each member has one role per project, compared by privilege (`owner > editor > viewer`):

| Role | Can |
|---|---|
| **viewer** | read configs, values, versions, the project's audit feed |
| **editor** | everything a viewer can, plus write configs/values/files, clone, manage environments and service tokens |
| **owner** | everything an editor can, plus manage **members** and **invitations** |

Enforcement is at a single choke point on every `/projects/{project}/*` route: a **non-member gets 404** (the
project's existence is hidden), and a **member below the required role gets 403**. A project always keeps **at
least one owner**.

## Authentication planes

| Plane | Credential | Use |
|---|---|---|
| **Session** | `otdm_session` cookie (+ CSRF) | the web UI |
| **User PAT** | `otdmu_…` Bearer | acts *as the user* for CLI/management writes; inherits the user's roles |
| **Service token** | `otdm_…` Bearer | **read-only**, scoped to one project + explicit environments; for CI [resolve](/guide/ci) |

A service token is its own grant (independent of any user's membership); a PAT is the user. Mint service tokens
in a project's **Settings → Service tokens**; manage your PATs under **Tokens**.

## Invitations

New users join by **email invitation**:

1. An **owner** (or admin) invites an email with a role in **Settings → Members → Invite**.
2. The invitee opens the accept link, picks a username + password, and is logged in with that role.

If SMTP is [configured](/guide/configuration#smtp-optional-for-invitation-emails) the link is emailed; otherwise
it is written to the server logs and shown to the inviter. Invitation tokens are hashed at rest, single-use, and
expire after 7 days. Existing users are added directly by username/email (no email step).

## Admin user directory

Instance admins get a **Users** page to list users, grant/revoke admin, and deactivate accounts (a deactivated
user's existing sessions stop working immediately). The instance always keeps at least one active admin.

See [`DECISIONS.md`](https://github.com/opentdm/opentdm/blob/main/DECISIONS.md) for the binding details.
