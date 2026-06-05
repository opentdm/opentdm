-- Projects (with envelope-encryption key columns), environments, configs,
-- config items (the variable merge unit), and scoped service tokens.

CREATE TABLE projects (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         uuid,                       -- NULL in v1; multi-tenant seam
    slug           citext NOT NULL,
    name           text   NOT NULL,
    description    text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    dek_wrapped    bytea  NOT NULL,            -- per-project DEK, KEK-wrapped
    dek_key_ref    text   NOT NULL,            -- which KEK wrapped it (rotation)
    dek_version    integer NOT NULL DEFAULT 1,
    crypto_version smallint NOT NULL DEFAULT 1,
    archived_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
-- v1: slug unique instance-wide (org_id IS NULL). When orgs land, swap for a
-- per-org unique index.
CREATE UNIQUE INDEX uq_projects_slug_global ON projects(slug) WHERE org_id IS NULL;
CREATE TRIGGER trg_projects_updated BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE environments (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        citext NOT NULL,
    name        text   NOT NULL,
    rank        integer NOT NULL DEFAULT 0,     -- ordering: dev < staging < prod
    is_default  boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT environments_project_slug_key UNIQUE (project_id, slug)
);
CREATE INDEX idx_environments_project ON environments(project_id);
CREATE TRIGGER trg_environments_updated BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE configs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind        config_kind   NOT NULL,
    format      config_format NOT NULL,
    name        text NOT NULL,
    sort_order  integer NOT NULL DEFAULT 0,     -- deterministic merge precedence
    description text,
    is_secret   boolean NOT NULL DEFAULT false,
    archived_at timestamptz,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT configs_project_name_key UNIQUE (project_id, name),
    CONSTRAINT configs_kind_format_chk CHECK (
        (kind = 'variable' AND format IN ('env','properties','secret'))
     OR (kind = 'file'     AND format IN ('json','csv','xml'))
    )
);
CREATE INDEX idx_configs_project ON configs(project_id);
CREATE TRIGGER trg_configs_updated BEFORE UPDATE ON configs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE config_tags (
    config_id uuid NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
    tag       citext NOT NULL,
    PRIMARY KEY (config_id, tag)
);
CREATE INDEX idx_config_tags_tag ON config_tags(tag);

-- Variable items: base = env_id NULL; override = env_id set. value_ciphertext
-- holds the full versioned envelope ([alg][nonce][ct||tag]); deleted=true is the
-- env-level tombstone.
CREATE TABLE config_items (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id        uuid NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
    env_id           uuid REFERENCES environments(id) ON DELETE CASCADE,
    key              text  NOT NULL,
    value_ciphertext bytea NOT NULL,
    dek_version      integer NOT NULL,
    is_secret        boolean NOT NULL DEFAULT false,
    deleted          boolean NOT NULL DEFAULT false,
    updated_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
-- Paired partial unique indexes: NULL is distinct in a multi-col UNIQUE, so the
-- base layer (env_id IS NULL) needs its own guard.
CREATE UNIQUE INDEX uq_config_items_base ON config_items(config_id, key) WHERE env_id IS NULL;
CREATE UNIQUE INDEX uq_config_items_env  ON config_items(config_id, env_id, key) WHERE env_id IS NOT NULL;
CREATE INDEX idx_config_items_config_env ON config_items(config_id, env_id);
CREATE TRIGGER trg_config_items_updated BEFORE UPDATE ON config_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE api_tokens (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         text NOT NULL,
    token_prefix text NOT NULL,
    token_hash   bytea NOT NULL UNIQUE,
    scope        token_scope NOT NULL DEFAULT 'read',
    expires_at   timestamptz,
    last_used_at timestamptz,
    last_used_ip inet,
    revoked_at   timestamptz,
    created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_tokens_project ON api_tokens(project_id);

-- A token is scoped to one project and >= 1 explicit environments (default-deny).
CREATE TABLE api_token_environments (
    token_id       uuid NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    environment_id uuid NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    PRIMARY KEY (token_id, environment_id)
);
CREATE INDEX idx_token_envs_env ON api_token_environments(environment_id);
