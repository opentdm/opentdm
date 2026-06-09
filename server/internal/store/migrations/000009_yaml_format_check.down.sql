ALTER TABLE configs DROP CONSTRAINT configs_kind_format_chk;
ALTER TABLE configs ADD CONSTRAINT configs_kind_format_chk CHECK (
    (kind = 'variable' AND format IN ('env','properties','secret'))
 OR (kind = 'file'     AND format IN ('json','csv','xml'))
);
