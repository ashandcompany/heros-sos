DROP TRIGGER hero_stats_after_truncate ON intervention_heroes;
DROP TRIGGER hero_stats_after_delete ON intervention_heroes;
DROP TRIGGER hero_stats_after_update ON intervention_heroes;
DROP TRIGGER hero_stats_after_insert ON intervention_heroes;
DROP FUNCTION hero_stats_after_truncate();
DROP FUNCTION hero_stats_after_delete();
DROP FUNCTION hero_stats_after_update();
DROP FUNCTION hero_stats_after_insert();

DROP TRIGGER villain_stats_after_truncate ON incident_villains;
DROP TRIGGER villain_stats_after_delete ON incident_villains;
DROP TRIGGER villain_stats_after_insert ON incident_villains;
DROP FUNCTION villain_stats_after_truncate();
DROP FUNCTION villain_stats_after_delete();
DROP FUNCTION villain_stats_after_insert();

DROP TRIGGER incident_stats_after_truncate ON incidents;
DROP TRIGGER incident_stats_after_delete ON incidents;
DROP TRIGGER incident_stats_after_update ON incidents;
DROP TRIGGER incident_stats_after_insert ON incidents;
DROP FUNCTION incident_stats_after_truncate();
DROP FUNCTION incident_stats_after_delete();
DROP FUNCTION incident_stats_after_update();
DROP FUNCTION incident_stats_after_insert();

DROP FUNCTION rebuild_read_counters();
DROP TABLE hero_stats;
DROP TABLE villain_stats;
DROP TABLE reporter_stats;
DROP TABLE incident_stats;
