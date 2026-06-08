-- Recreate config_tags (reverses 000006). The citext extension is created earlier
-- and is unaffected by the drop, so the type resolves here.
CREATE TABLE config_tags (
    config_id uuid NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
    tag       citext NOT NULL,
    PRIMARY KEY (config_id, tag)
);
CREATE INDEX idx_config_tags_tag ON config_tags(tag);
