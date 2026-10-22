ALTER TABLE incidents
    ADD COLUMN moderation text NOT NULL DEFAULT 'approved',
    ADD COLUMN reporter_role text,
    DROP CONSTRAINT incidents_priority_check,
    ADD CONSTRAINT incidents_priority_check
        CHECK (priority IN ('high', 'normal', 'low', 'critical')),
    ADD CONSTRAINT incidents_moderation_check
        CHECK (moderation IN ('pending', 'approved', 'rejected')),
    ADD CONSTRAINT incidents_reporter_role_check
        CHECK (reporter_role IS NULL OR reporter_role IN ('witness', 'victim'));

ALTER TABLE incidents ALTER COLUMN moderation SET DEFAULT 'pending';
