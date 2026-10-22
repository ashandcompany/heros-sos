-- Date de résolution d'un incident, recopiée sur la ligne de l'incident : la fin de sa dernière intervention (la plus
-- récemment démarrée), s'il est réussi ; NULL sinon.
--
-- GET /incidents?status=succeeded&sort=-resolvedAt (« les derniers incidents résolus », la page d'accueil, sans
-- session) la calculait pour chaque incident de la table, par une jointure sur sa dernière intervention, avant de
-- trier : un parcours complet à chaque affichage. Avec la colonne et son index (add_incident_resolved_index), la page
-- est lue dans l'ordre de l'index, comme les autres listes.
--
-- Placée après add_incident_perf_indexes : le rattrapage ci-dessous cherche la dernière intervention de chaque incident
-- réussi avec interventions_incident_idx ; sans cet index il parcourrait toute la table pour chacun.
--
-- Tenue par des triggers de niveau instruction (une requête quel que soit le nombre de lignes écrites) : aucun
-- chemin d'écriture (API, correction par la modération, seed, SQL à la main) ne peut l'oublier.

ALTER TABLE incidents ADD COLUMN resolved_at timestamptz;

-- La référence du calcul, partagée par le rattrapage et les triggers.
CREATE FUNCTION incident_resolved_at(p_incident uuid, p_status text) RETURNS timestamptz
LANGUAGE sql STABLE AS $$
    SELECT CASE WHEN p_status = 'succeeded' THEN (
        SELECT ended_at FROM interventions
         WHERE incident_id = p_incident
         ORDER BY started_at DESC NULLS LAST, id DESC
         LIMIT 1) END
$$;

UPDATE incidents SET resolved_at = incident_resolved_at(id, status) WHERE status = 'succeeded';

-- Un incident change de statut : sa date de résolution apparaît ou disparaît.
--
-- Un trigger de niveau instruction se déclenche même quand l'UPDATE ne touche aucune ligne, et celui-ci écrit dans
-- incidents : sans la garde ci-dessous, son propre UPDATE le redéclencherait à l'infini (stack depth limit exceeded).
-- Rien à faire sans changement de statut ; l'UPDATE interne ne change que resolved_at, il passe donc la garde une fois.
CREATE FUNCTION incidents_sync_resolved_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM new_rows n JOIN old_rows o ON o.id = n.id WHERE n.status IS DISTINCT FROM o.status) THEN
        RETURN NULL;
    END IF;
    UPDATE incidents
       SET resolved_at = changed.resolved
      FROM (SELECT incidents.id, incident_resolved_at(incidents.id, incidents.status) AS resolved
              FROM incidents
              JOIN new_rows n ON n.id = incidents.id
              JOIN old_rows o ON o.id = n.id
             WHERE n.status IS DISTINCT FROM o.status) AS changed
     WHERE incidents.id = changed.id
       AND incidents.resolved_at IS DISTINCT FROM changed.resolved;
    RETURN NULL;
END $$;

CREATE TRIGGER incidents_sync_resolved_at
    AFTER UPDATE ON incidents
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION incidents_sync_resolved_at();

-- Une intervention est créée, ou son statut, son début ou sa fin changent : son incident recalcule sa date. L'incident
-- passe en général au statut « réussi » dans la même transaction, par un autre ordre : les deux triggers
-- convergent, quel que soit l'ordre dans lequel ils s'exécutent. Une écriture qui ne change rien n'écrit rien (un
-- lot d'incidents inséré avec sa date, comme cmd/seedperf, ne réécrit aucune ligne).
CREATE FUNCTION interventions_sync_incident_resolved_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE incidents
       SET resolved_at = changed.resolved
      FROM (SELECT incidents.id, incident_resolved_at(incidents.id, incidents.status) AS resolved
              FROM incidents
             WHERE incidents.id IN (SELECT incident_id FROM new_rows)) AS changed
     WHERE incidents.id = changed.id
       AND incidents.resolved_at IS DISTINCT FROM changed.resolved;
    RETURN NULL;
END $$;

CREATE TRIGGER interventions_sync_incident_resolved_at_insert
    AFTER INSERT ON interventions
    REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION interventions_sync_incident_resolved_at();

CREATE TRIGGER interventions_sync_incident_resolved_at_update
    AFTER UPDATE ON interventions
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION interventions_sync_incident_resolved_at();
