DROP TRIGGER incidents_sync_moderation ON incidents;
DROP FUNCTION incidents_sync_moderation();

DROP TRIGGER interventions_sync_resolved_at ON interventions;
DROP FUNCTION interventions_sync_resolved_at();

DROP TRIGGER intervention_heroes_fill_resolved_at ON intervention_heroes;
DROP FUNCTION intervention_heroes_fill_resolved_at();
ALTER TABLE intervention_heroes DROP COLUMN resolved_at;

DROP TRIGGER incident_villains_fill_from_incident ON incident_villains;
DROP FUNCTION incident_villains_fill_from_incident();
ALTER TABLE incident_villains DROP COLUMN incident_approved;
ALTER TABLE incident_villains DROP COLUMN incident_created_at;
