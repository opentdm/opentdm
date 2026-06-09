-- Add 'yaml' to the config_format enum. Must land in its own migration: Postgres
-- forbids using a newly-added enum value in the same transaction, and 000009
-- references it in the kind/format CHECK.
ALTER TYPE config_format ADD VALUE IF NOT EXISTS 'yaml';
