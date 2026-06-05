-- Phase 2: file/fixture current-state blobs, append-only version history (both
-- kinds), and user Personal Access Tokens.

-- File current-state (mirrors config_items; env_id NULL = default variant).
CREATE TABLE config_blobs (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id          uuid NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
    env_id             uuid REFERENCES environments(id) ON DELETE CASCADE,
    content_ciphertext bytea   NOT NULL,            -- [alg][nonce][ct||tag], sealed with BlobAAD
    dek_version        integer NOT NULL,
    content_hmac       bytea   NOT NULL,            -- HMAC(HKDF(DEK), plaintext)
    size_bytes         bigint  NOT NULL,
    updated_by         uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_config_blobs_base ON config_blobs(config_id)         WHERE env_id IS NULL;
CREATE UNIQUE INDEX uq_config_blobs_env  ON config_blobs(config_id, env_id) WHERE env_id IS NOT NULL;
CREATE TRIGGER trg_config_blobs_updated BEFORE UPDATE ON config_blobs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Append-only history for BOTH kinds. A version is ONE row: the whole layer
-- snapshot sealed as a single blob (canonical JSON of items for variables; raw
-- plaintext for files).
CREATE TABLE config_versions (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id           uuid NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
    env_id              uuid REFERENCES environments(id) ON DELETE CASCADE,  -- NULL = base/default layer
    version             integer NOT NULL,                -- monotonic per (config, env-or-base)
    snapshot_kind       config_kind NOT NULL,            -- denormalized from configs.kind
    snapshot_ciphertext bytea   NOT NULL,                -- sealed with VersionAAD
    dek_version         integer NOT NULL,                -- DEK gen used to seal THIS row
    content_hmac        bytea   NOT NULL,                -- HMAC(HKDF(DEK), canonical plaintext)
    byte_size           bigint  NOT NULL,
    is_current          boolean NOT NULL DEFAULT true,
    comment             text,
    created_by          uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now()
);
-- Monotonic numbering per layer (NULL-base gotcha: NULL is distinct in multi-col UNIQUE).
CREATE UNIQUE INDEX uq_config_versions_base_no ON config_versions(config_id, version)         WHERE env_id IS NULL;
CREATE UNIQUE INDEX uq_config_versions_env_no  ON config_versions(config_id, env_id, version) WHERE env_id IS NOT NULL;
-- Exactly one current per layer (a second current row is a DB-level error).
CREATE UNIQUE INDEX uq_config_versions_current_base ON config_versions(config_id)         WHERE env_id IS NULL     AND is_current;
CREATE UNIQUE INDEX uq_config_versions_current_env  ON config_versions(config_id, env_id) WHERE env_id IS NOT NULL AND is_current;
CREATE INDEX idx_config_versions_layer ON config_versions(config_id, env_id, version DESC);

-- User Personal Access Tokens: user-scoped (grant the user's management access),
-- distinct from project+env service tokens.
CREATE TABLE user_pats (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text  NOT NULL,
    token_prefix text  NOT NULL,                   -- "otdmu_" + display chars
    token_hash   bytea NOT NULL UNIQUE,            -- HMAC-SHA256(pepper, raw)
    expires_at   timestamptz,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_pats_user_name_key UNIQUE (user_id, name)
);
CREATE INDEX idx_user_pats_user ON user_pats(user_id);
