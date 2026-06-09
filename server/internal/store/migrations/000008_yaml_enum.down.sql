-- Postgres has no DROP VALUE for enums; the 'yaml' member is left in place.
-- (000009's down restores the CHECK so file/yaml rows can no longer be created.)
SELECT 1;
