DROP TRIGGER interventions_sync_resolved_at ON interventions;
DROP FUNCTION interventions_sync_resolved_at();

DROP TRIGGER intervention_heroes_fill_resolved_at ON intervention_heroes;
DROP FUNCTION intervention_heroes_fill_resolved_at();
ALTER TABLE intervention_heroes DROP COLUMN resolved_at;

DROP TRIGGER incident_villains_fill_created_at ON incident_villains;
DROP FUNCTION incident_villains_fill_created_at();
ALTER TABLE incident_villains DROP COLUMN incident_created_at;
