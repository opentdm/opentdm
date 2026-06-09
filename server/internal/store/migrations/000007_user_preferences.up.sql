-- Per-user UI preferences (theme + favourite project slugs). Client-only state
-- until now (localStorage); this makes it follow the account across devices.
ALTER TABLE users ADD COLUMN preferences jsonb NOT NULL DEFAULT '{}'::jsonb;
