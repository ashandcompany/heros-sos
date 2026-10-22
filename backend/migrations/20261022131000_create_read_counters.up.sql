-- Compteurs de lecture : les totaux et les statistiques que les listes affichent, tenus à jour à chaque écriture
-- au lieu d'être recalculés sur toute la table à chaque lecture (voir backend/stack/performance-et-synchronisation.md).
-- À 5 millions d'incidents, compter coûtait de 150 ms à plusieurs secondes selon le filtre ; lire un compteur
-- coûte une lecture d'index.
--
--   incident_stats : incidents et citoyens secourus par (statut, type)   -> total des listes, statistiques
--   reporter_stats : incidents de chaque déclarant, par statut           -> total de « mes incidents »
--   villain_stats  : incidents liés à chaque vilain                      -> fiche et liste des vilains
--   hero_stats     : interventions confirmées et résolues de chaque héros -> fiche et liste des héros
--
-- Ils sont tenus à jour par des triggers de niveau instruction (REFERENCING … TABLE) : une écriture en masse
-- (100 000 lignes) ne produit qu'une mise à jour par compteur, pas une par ligne, et aucun chemin d'écriture (API,
-- seed, SQL à la main) ne peut les oublier. rebuild_read_counters() les recalcule depuis les tables : c'est la
-- référence du calcul, utilisée ici pour le rattrapage initial, et pour corriger une dérive éventuelle.

CREATE TABLE incident_stats (
    status           text     NOT NULL,
    type_id          smallint NOT NULL REFERENCES incident_types (id),
    incidents        bigint   NOT NULL DEFAULT 0,
    citizens_rescued bigint   NOT NULL DEFAULT 0,

    CONSTRAINT incident_stats_pk PRIMARY KEY (status, type_id)
);

-- Sans clé étrangère vers users : le déclarant d'un incident passe à NULL quand son compte est supprimé, et la
-- mise à jour qui le fait décrémente sa ligne, qu'elle existe encore ou non.
CREATE TABLE reporter_stats (
    reporter_id uuid   NOT NULL,
    status      text   NOT NULL,
    incidents   bigint NOT NULL DEFAULT 0,

    CONSTRAINT reporter_stats_pk PRIMARY KEY (reporter_id, status)
);

CREATE TABLE villain_stats (
    villain_id uuid   PRIMARY KEY REFERENCES villains (id) ON DELETE CASCADE,
    incidents  bigint NOT NULL DEFAULT 0
);

CREATE TABLE hero_stats (
    hero_id       uuid   PRIMARY KEY REFERENCES hero_profiles (user_id) ON DELETE CASCADE,
    interventions bigint NOT NULL DEFAULT 0,
    resolved      bigint NOT NULL DEFAULT 0
);

CREATE FUNCTION rebuild_read_counters() RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE incident_stats, reporter_stats, villain_stats, hero_stats;

    INSERT INTO incident_stats (status, type_id, incidents, citizens_rescued)
    SELECT coalesce(status, 'reported'), type_id, count(*), coalesce(sum(citizens_rescued), 0)
      FROM incidents
     GROUP BY coalesce(status, 'reported'), type_id;

    INSERT INTO reporter_stats (reporter_id, status, incidents)
    SELECT reporter_id, coalesce(status, 'reported'), count(*)
      FROM incidents
     WHERE reporter_id IS NOT NULL
     GROUP BY reporter_id, coalesce(status, 'reported');

    INSERT INTO villain_stats (villain_id, incidents)
    SELECT villain_id, count(*) FROM incident_villains GROUP BY villain_id;

    INSERT INTO hero_stats (hero_id, interventions, resolved)
    SELECT hero_id, count(*), count(resolved_at)
      FROM intervention_heroes
     WHERE participation = 'confirmed'
     GROUP BY hero_id;
END $$;

SELECT rebuild_read_counters();

-- --- incidents -> incident_stats -----------------------------------------------------------------------------

CREATE FUNCTION incident_stats_after_insert() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO incident_stats AS s (status, type_id, incidents, citizens_rescued)
    SELECT coalesce(status, 'reported'), type_id, count(*), coalesce(sum(citizens_rescued), 0)
      FROM new_rows
     GROUP BY coalesce(status, 'reported'), type_id
    ON CONFLICT (status, type_id) DO UPDATE
        SET incidents = s.incidents + EXCLUDED.incidents,
            citizens_rescued = s.citizens_rescued + EXCLUDED.citizens_rescued;

    INSERT INTO reporter_stats AS s (reporter_id, status, incidents)
    SELECT reporter_id, coalesce(status, 'reported'), count(*)
      FROM new_rows
     WHERE reporter_id IS NOT NULL
     GROUP BY reporter_id, coalesce(status, 'reported')
    ON CONFLICT (reporter_id, status) DO UPDATE SET incidents = s.incidents + EXCLUDED.incidents;
    RETURN NULL;
END $$;

-- Une mise à jour qui ne change ni le statut, ni le type, ni les citoyens secourus (updated_at seul, par exemple)
-- donne des deltas nuls : rien n'est écrit.
CREATE FUNCTION incident_stats_after_update() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO incident_stats AS s (status, type_id, incidents, citizens_rescued)
    SELECT status, type_id, sum(n), sum(rescued)
      FROM (SELECT coalesce(status, 'reported') AS status, type_id, 1 AS n, coalesce(citizens_rescued, 0)::bigint AS rescued
              FROM new_rows
            UNION ALL
            SELECT coalesce(status, 'reported'), type_id, -1, -coalesce(citizens_rescued, 0)::bigint
              FROM old_rows) AS changes
     GROUP BY status, type_id
    HAVING sum(n) <> 0 OR sum(rescued) <> 0
    ON CONFLICT (status, type_id) DO UPDATE
        SET incidents = s.incidents + EXCLUDED.incidents,
            citizens_rescued = s.citizens_rescued + EXCLUDED.citizens_rescued;

    -- Déclarant : les retraits (statut quitté, compte supprimé) ne touchent que des lignes existantes ;
    -- les ajouts créent la ligne au besoin. Une même clé est soit retirée, soit ajoutée, jamais les deux.
    WITH net AS (
        SELECT reporter_id, status, sum(n) AS n
          FROM (SELECT reporter_id, coalesce(status, 'reported') AS status, 1 AS n FROM new_rows WHERE reporter_id IS NOT NULL
                UNION ALL
                SELECT reporter_id, coalesce(status, 'reported'), -1 FROM old_rows WHERE reporter_id IS NOT NULL) AS changes
         GROUP BY reporter_id, status
        HAVING sum(n) <> 0
    ), removed AS (
        UPDATE reporter_stats AS s SET incidents = s.incidents + net.n
          FROM net
         WHERE s.reporter_id = net.reporter_id AND s.status = net.status AND net.n < 0
    )
    INSERT INTO reporter_stats AS s (reporter_id, status, incidents)
    SELECT reporter_id, status, n FROM net WHERE n > 0
    ON CONFLICT (reporter_id, status) DO UPDATE SET incidents = s.incidents + EXCLUDED.incidents;
    RETURN NULL;
END $$;

CREATE FUNCTION incident_stats_after_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE incident_stats AS s
       SET incidents = s.incidents - d.incidents,
           citizens_rescued = s.citizens_rescued - d.rescued
      FROM (SELECT coalesce(status, 'reported') AS status, type_id, count(*) AS incidents,
                   coalesce(sum(citizens_rescued), 0) AS rescued
              FROM old_rows
             GROUP BY coalesce(status, 'reported'), type_id) AS d
     WHERE s.status = d.status AND s.type_id = d.type_id;

    UPDATE reporter_stats AS s
       SET incidents = s.incidents - d.incidents
      FROM (SELECT reporter_id, coalesce(status, 'reported') AS status, count(*) AS incidents
              FROM old_rows
             WHERE reporter_id IS NOT NULL
             GROUP BY reporter_id, coalesce(status, 'reported')) AS d
     WHERE s.reporter_id = d.reporter_id AND s.status = d.status;
    RETURN NULL;
END $$;

-- TRUNCATE ne déclenche pas les triggers de ligne : les compteurs sont vidés avec la table.
CREATE FUNCTION incident_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE incident_stats, reporter_stats;
    RETURN NULL;
END $$;

CREATE TRIGGER incident_stats_after_insert AFTER INSERT ON incidents
    REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION incident_stats_after_insert();
CREATE TRIGGER incident_stats_after_update AFTER UPDATE ON incidents
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION incident_stats_after_update();
CREATE TRIGGER incident_stats_after_delete AFTER DELETE ON incidents
    REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION incident_stats_after_delete();
CREATE TRIGGER incident_stats_after_truncate AFTER TRUNCATE ON incidents
    FOR EACH STATEMENT EXECUTE FUNCTION incident_stats_after_truncate();

-- --- incident_villains -> villain_stats ----------------------------------------------------------------------

CREATE FUNCTION villain_stats_after_insert() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO villain_stats AS s (villain_id, incidents)
    SELECT villain_id, count(*) FROM new_rows GROUP BY villain_id
    ON CONFLICT (villain_id) DO UPDATE SET incidents = s.incidents + EXCLUDED.incidents;
    RETURN NULL;
END $$;

CREATE FUNCTION villain_stats_after_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE villain_stats AS s
       SET incidents = s.incidents - d.incidents
      FROM (SELECT villain_id, count(*) AS incidents FROM old_rows GROUP BY villain_id) AS d
     WHERE s.villain_id = d.villain_id;
    RETURN NULL;
END $$;

CREATE FUNCTION villain_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE villain_stats;
    RETURN NULL;
END $$;

CREATE TRIGGER villain_stats_after_insert AFTER INSERT ON incident_villains
    REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION villain_stats_after_insert();
CREATE TRIGGER villain_stats_after_delete AFTER DELETE ON incident_villains
    REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION villain_stats_after_delete();
CREATE TRIGGER villain_stats_after_truncate AFTER TRUNCATE ON incident_villains
    FOR EACH STATEMENT EXECUTE FUNCTION villain_stats_after_truncate();

-- --- intervention_heroes -> hero_stats -----------------------------------------------------------------------

CREATE FUNCTION hero_stats_after_insert() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO hero_stats AS s (hero_id, interventions, resolved)
    SELECT hero_id, count(*), count(resolved_at)
      FROM new_rows
     WHERE participation = 'confirmed'
     GROUP BY hero_id
    ON CONFLICT (hero_id) DO UPDATE
        SET interventions = s.interventions + EXCLUDED.interventions,
            resolved = s.resolved + EXCLUDED.resolved;
    RETURN NULL;
END $$;

-- Confirmation d'une invitation, ou date de résolution posée à la clôture : le delta est celui des lignes
-- confirmées avant et après.
CREATE FUNCTION hero_stats_after_update() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO hero_stats AS s (hero_id, interventions, resolved)
    SELECT hero_id, sum(n), sum(r)
      FROM (SELECT hero_id, 1 AS n, (resolved_at IS NOT NULL)::int AS r
              FROM new_rows WHERE participation = 'confirmed'
            UNION ALL
            SELECT hero_id, -1, -(resolved_at IS NOT NULL)::int
              FROM old_rows WHERE participation = 'confirmed') AS changes
     GROUP BY hero_id
    HAVING sum(n) <> 0 OR sum(r) <> 0
    ON CONFLICT (hero_id) DO UPDATE
        SET interventions = s.interventions + EXCLUDED.interventions,
            resolved = s.resolved + EXCLUDED.resolved;
    RETURN NULL;
END $$;

CREATE FUNCTION hero_stats_after_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE hero_stats AS s
       SET interventions = s.interventions - d.interventions,
           resolved = s.resolved - d.resolved
      FROM (SELECT hero_id, count(*) AS interventions, count(resolved_at) AS resolved
              FROM old_rows
             WHERE participation = 'confirmed'
             GROUP BY hero_id) AS d
     WHERE s.hero_id = d.hero_id;
    RETURN NULL;
END $$;

CREATE FUNCTION hero_stats_after_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE hero_stats;
    RETURN NULL;
END $$;

CREATE TRIGGER hero_stats_after_insert AFTER INSERT ON intervention_heroes
    REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION hero_stats_after_insert();
CREATE TRIGGER hero_stats_after_update AFTER UPDATE ON intervention_heroes
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION hero_stats_after_update();
CREATE TRIGGER hero_stats_after_delete AFTER DELETE ON intervention_heroes
    REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION hero_stats_after_delete();
CREATE TRIGGER hero_stats_after_truncate AFTER TRUNCATE ON intervention_heroes
    FOR EACH STATEMENT EXECUTE FUNCTION hero_stats_after_truncate();
