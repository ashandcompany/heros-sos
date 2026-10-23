-- Les triggers AFTER TRUNCATE des compteurs de lecture vidaient les compteurs par TRUNCATE. Or un
-- TRUNCATE … CASCADE (cmd/seed -reset) inclut déjà ces compteurs dans l'instruction, par leurs clés étrangères
-- vers incident_types, villains et hero_profiles : le TRUNCATE du trigger échouait alors avec
-- « cannot TRUNCATE "incident_stats" because it is being used by active queries in this session ».
-- DELETE n'a pas cette restriction, et les compteurs restent de petites tables.

CREATE OR REPLACE FUNCTION incident_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM incident_stats;
    DELETE FROM reporter_stats;
    RETURN NULL;
END $$;

CREATE OR REPLACE FUNCTION villain_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM villain_stats;
    RETURN NULL;
END $$;

CREATE OR REPLACE FUNCTION hero_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM hero_stats;
    RETURN NULL;
END $$;
