-- Colonnes recopiées d'une autre table, pour qu'un index puisse ordonner les historiques publics d'un vilain et
-- d'un héros (voir backend/stack/performance-et-synchronisation.md). Sans elles, lister les incidents d'un vilain
-- qui en a 125 000 obligeait à les lire tous pour les trier par date.
--
-- Les historiques publics ne montrent que les incidents approuvés par la modération (incidents.moderation) : cette
-- information est recopiée avec la date, pour que l'index serve directement la page et que les compteurs de
-- lecture (create_read_counters) la connaissent sans jointure.
--
-- Toutes les colonnes sont remplies par des triggers : aucun chemin d'écriture (API, seed, SQL à la main) ne peut
-- les oublier, et une décision de modération les met à jour toutes.

-- Date de création de l'incident lié (immuable) et indicateur « approuvé » (change avec la modération).
ALTER TABLE incident_villains ADD COLUMN incident_created_at timestamptz;
ALTER TABLE incident_villains ADD COLUMN incident_approved boolean;

UPDATE incident_villains
   SET incident_created_at = incidents.created_at,
       incident_approved = (incidents.moderation = 'approved')
  FROM incidents
 WHERE incidents.id = incident_villains.incident_id;

ALTER TABLE incident_villains
    ALTER COLUMN incident_created_at SET NOT NULL,
    ALTER COLUMN incident_approved SET NOT NULL;

CREATE FUNCTION incident_villains_fill_from_incident() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    SELECT created_at, moderation = 'approved'
      INTO NEW.incident_created_at, NEW.incident_approved
      FROM incidents
     WHERE id = NEW.incident_id;
    RETURN NEW;
END $$;

-- Les écritures en masse (cmd/seedperf) fournissent déjà les valeurs : le trigger ne se déclenche pas pour elles.
CREATE TRIGGER incident_villains_fill_from_incident
    BEFORE INSERT ON incident_villains
    FOR EACH ROW WHEN (NEW.incident_created_at IS NULL OR NEW.incident_approved IS NULL)
    EXECUTE FUNCTION incident_villains_fill_from_incident();

-- Date de résolution d'un héros : la fin de l'intervention réussie à laquelle il a participé (confirmé), pour un
-- incident approuvé. NULL tant que l'intervention n'a pas réussi, si le héros n'est qu'invité, ou si la
-- modération n'a pas (ou plus) approuvé l'incident : l'historique public et son compteur l'ignorent alors.
ALTER TABLE intervention_heroes ADD COLUMN resolved_at timestamptz;

UPDATE intervention_heroes
   SET resolved_at = coalesce(interventions.ended_at, interventions.started_at, now())
  FROM interventions
  JOIN incidents ON incidents.id = interventions.incident_id
 WHERE interventions.id = intervention_heroes.intervention_id
   AND interventions.status = 'succeeded'
   AND incidents.moderation = 'approved'
   AND intervention_heroes.participation = 'confirmed';

-- Un héros confirmé sur une intervention déjà réussie (ou qui confirme son invitation) reçoit la date.
CREATE FUNCTION intervention_heroes_fill_resolved_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    SELECT CASE WHEN interventions.status = 'succeeded' AND incidents.moderation = 'approved'
                THEN coalesce(interventions.ended_at, interventions.started_at, now()) END
      INTO NEW.resolved_at
      FROM interventions
      JOIN incidents ON incidents.id = interventions.incident_id
     WHERE interventions.id = NEW.intervention_id;
    RETURN NEW;
END $$;

CREATE TRIGGER intervention_heroes_fill_resolved_at
    BEFORE INSERT OR UPDATE OF participation ON intervention_heroes
    FOR EACH ROW WHEN (NEW.participation = 'confirmed' AND NEW.resolved_at IS NULL)
    EXECUTE FUNCTION intervention_heroes_fill_resolved_at();

-- Quand une intervention réussit (ou cesse de l'être), ses héros confirmés suivent. Niveau instruction : une
-- seule requête quel que soit le nombre d'interventions modifiées.
CREATE FUNCTION interventions_sync_resolved_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE intervention_heroes
       SET resolved_at = CASE WHEN n.status = 'succeeded' AND incidents.moderation = 'approved'
                              THEN coalesce(n.ended_at, n.started_at, now()) END
      FROM new_rows n
      JOIN old_rows o ON o.id = n.id
      JOIN incidents ON incidents.id = n.incident_id
     WHERE intervention_heroes.intervention_id = n.id
       AND intervention_heroes.participation = 'confirmed'
       AND (n.status IS DISTINCT FROM o.status OR n.ended_at IS DISTINCT FROM o.ended_at)
       AND intervention_heroes.resolved_at IS DISTINCT FROM
           CASE WHEN n.status = 'succeeded' AND incidents.moderation = 'approved'
                THEN coalesce(n.ended_at, n.started_at, now()) END;
    RETURN NULL;
END $$;

CREATE TRIGGER interventions_sync_resolved_at
    AFTER UPDATE ON interventions
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION interventions_sync_resolved_at();

-- Une décision de modération (PUT /admin/incidents/{id}/moderation) change ce que l'incident montre au public : ses
-- liens vers les vilains et les dates de résolution de ses héros suivent. Les compteurs (create_read_counters)
-- réagissent à ces mises à jour comme à n'importe quelle autre. Un trigger de niveau instruction ne peut pas être
-- limité à une colonne : les mises à jour qui ne touchent pas la modération ne trouvent simplement rien à faire.
CREATE FUNCTION incidents_sync_moderation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE incident_villains
       SET incident_approved = (n.moderation = 'approved')
      FROM new_rows n
      JOIN old_rows o ON o.id = n.id
     WHERE incident_villains.incident_id = n.id
       AND n.moderation IS DISTINCT FROM o.moderation
       AND incident_villains.incident_approved IS DISTINCT FROM (n.moderation = 'approved');

    UPDATE intervention_heroes
       SET resolved_at = CASE WHEN interventions.status = 'succeeded' AND n.moderation = 'approved'
                              THEN coalesce(interventions.ended_at, interventions.started_at, now()) END
      FROM new_rows n
      JOIN old_rows o ON o.id = n.id
      JOIN interventions ON interventions.incident_id = n.id
     WHERE intervention_heroes.intervention_id = interventions.id
       AND intervention_heroes.participation = 'confirmed'
       AND n.moderation IS DISTINCT FROM o.moderation
       AND intervention_heroes.resolved_at IS DISTINCT FROM
           CASE WHEN interventions.status = 'succeeded' AND n.moderation = 'approved'
                THEN coalesce(interventions.ended_at, interventions.started_at, now()) END;
    RETURN NULL;
END $$;

CREATE TRIGGER incidents_sync_moderation
    AFTER UPDATE ON incidents
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION incidents_sync_moderation();
