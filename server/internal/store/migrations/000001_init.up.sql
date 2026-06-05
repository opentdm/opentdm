-- Extensions, enums, the shared updated_at trigger, users, sessions, and the
-- bootstrap singleton.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;   -- case-insensitive identifiers

CREATE TYPE config_kind   AS ENUM ('variable', 'file');
CREATE TYPE config_format AS ENUM ('env', 'properties', 'secret', 'json', 'csv', 'xml');
CREATE TYPE token_scope   AS ENUM ('read', 'write');

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username      citext NOT NULL UNIQUE,
    email         citext NOT NULL UNIQUE,
    password_hash text   NOT NULL,
    is_admin      boolean NOT NULL DEFAULT false,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   bytea NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user       ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Single-row table: the partial unique guard makes a second bootstrap insert
-- fail at the DB layer, closing the first-admin race.
CREATE TABLE setup_singleton (
    id      boolean PRIMARY KEY DEFAULT true CHECK (id),
    used_at timestamptz NOT NULL DEFAULT now()
);
