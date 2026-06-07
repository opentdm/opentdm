# Audit log

opentdm records an **append-only audit log** of resource mutations — who changed, cloned, invited, or tokened
what, and when — so a multi-user instance is accountable.

## What's recorded

Successful state-changing actions (config/value/file writes, rollback, clone, environment / member / invitation
/ token changes, project creation, admin user changes). Each entry captures the **actor**, a semantic **action**
(e.g. `config.items.updated`, `environment.cloned`, `member.added`), the **target**, status, IP, and timestamp.

**Never recorded:** request or response bodies — so **no secret values** ever land in the audit log. Auth
events (logins/logouts) are intentionally out of scope.

## Where to see it

- **Per project:** the **Activity** page — visible to any project member (viewer and up).
- **Instance-wide:** the **Activity** link in the header — instance admins only, spanning all projects.

Both feeds are newest-first with keyset pagination ("Load more"). Actor names are resolved for display and
remain readable even after a user is deleted.

This complements [version history](/guide/architecture) (which holds the per-config, per-version detail and
diffs) with the cross-cutting events that versions don't capture.
