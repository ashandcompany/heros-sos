UPDATE incidents SET priority = 'high' WHERE priority = 'critical';

ALTER TABLE incidents
    DROP CONSTRAINT incidents_reporter_role_check,
    DROP CONSTRAINT incidents_moderation_check,
    DROP CONSTRAINT incidents_priority_check,
    DROP COLUMN reporter_role,
    DROP COLUMN moderation,
    ADD CONSTRAINT incidents_priority_check CHECK (priority IN ('high', 'normal', 'low'));
