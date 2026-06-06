-- Per-project authorization: roles + membership, plus email invitations.

CREATE TYPE project_member_role AS ENUM ('owner', 'editor', 'viewer');

-- A user's role on a project. Instance admins (users.is_admin) bypass this and
-- have implicit owner access everywhere.
CREATE TABLE project_members (
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    role       project_member_role NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE TRIGGER trg_project_members_updated BEFORE UPDATE ON project_members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Email invitations: onboard a (new or existing) user onto a project with a
-- role. token_hash is HMAC(pepper, token) — single-use, expiring.
CREATE TABLE project_invitations (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email            citext NOT NULL,
    role             project_member_role NOT NULL,
    token_hash       bytea NOT NULL UNIQUE,
    invited_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    expires_at       timestamptz NOT NULL,
    accepted_at      timestamptz,
    accepted_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_project_invitations_project ON project_invitations(project_id);
CREATE INDEX idx_project_invitations_pending ON project_invitations(project_id) WHERE accepted_at IS NULL;

-- Backfill: the creator of each existing project becomes its owner.
INSERT INTO project_members (project_id, user_id, role)
SELECT id, created_by, 'owner' FROM projects WHERE created_by IS NOT NULL
ON CONFLICT DO NOTHING;
