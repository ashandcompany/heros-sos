CREATE OR REPLACE FUNCTION incident_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE incident_stats, reporter_stats;
    RETURN NULL;
END $$;

CREATE OR REPLACE FUNCTION villain_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE villain_stats;
    RETURN NULL;
END $$;

CREATE OR REPLACE FUNCTION hero_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE hero_stats;
    RETURN NULL;
END $$;
