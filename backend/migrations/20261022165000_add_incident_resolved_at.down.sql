DROP TRIGGER interventions_sync_incident_resolved_at_update ON interventions;
DROP TRIGGER interventions_sync_incident_resolved_at_insert ON interventions;
DROP FUNCTION interventions_sync_incident_resolved_at();

DROP TRIGGER incidents_sync_resolved_at ON incidents;
DROP FUNCTION incidents_sync_resolved_at();

DROP FUNCTION incident_resolved_at(uuid, text);
ALTER TABLE incidents DROP COLUMN resolved_at;
