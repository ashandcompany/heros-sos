ALTER TABLE villains
    ADD COLUMN threat_level text NOT NULL DEFAULT 'faible',
    ADD COLUMN status text NOT NULL DEFAULT 'en-liberte',
    ADD CONSTRAINT villains_threat_level_check
        CHECK (threat_level IN ('faible', 'modere', 'eleve', 'critique')),
    ADD CONSTRAINT villains_status_check
        CHECK (status IN ('en-liberte', 'emprisonne'));
