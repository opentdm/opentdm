-- Append-only audit log of resource mutations (who did what, when).

CREATE TABLE audit_log (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid REFERENCES projects(id) ON DELETE SET NULL,
    actor_user_id uuid REFERENCES users(id)    ON DELETE SET NULL,
    action        text NOT NULL,
    target_type   text,
    target_id     text,
    status        integer NOT NULL,
    ip            text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Keyset pagination indexes: per-project feed and the admin-wide feed.
CREATE INDEX idx_audit_project ON audit_log(project_id, created_at DESC, id DESC);
CREATE INDEX idx_audit_global  ON audit_log(created_at DESC, id DESC);
