ALTER TABLE villains
    DROP CONSTRAINT villains_status_check,
    DROP CONSTRAINT villains_threat_level_check,
    DROP COLUMN status,
    DROP COLUMN threat_level;
