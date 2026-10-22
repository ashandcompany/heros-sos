-- Deux colonnes recopiées d'une autre table, pour qu'un index puisse ordonner les historiques publics d'un
-- vilain et d'un héros (voir backend/stack/performance-et-synchronisation.md). Sans elles, lister les
-- incidents d'un vilain qui en a 125 000 obligeait à les lire tous pour les trier par date.
--
-- Les deux colonnes sont remplies par des triggers : aucun chemin d'écriture (API, seed, SQL à la main) ne
-- peut les oublier.

-- Date de création de l'incident lié (immuable : elle ne change jamais après la déclaration).
ALTER TABLE incident_villains ADD COLUMN incident_created_at timestamptz;

UPDATE incident_villains
   SET incident_created_at = incidents.created_at
  FROM incidents
 WHERE incidents.id = incident_villains.incident_id;

ALTER TABLE incident_villains ALTER COLUMN incident_created_at SET NOT NULL;

CREATE FUNCTION incident_villains_fill_created_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    SELECT created_at INTO NEW.incident_created_at FROM incidents WHERE id = NEW.incident_id;
    RETURN NEW;
END $$;

-- Les écritures en masse (cmd/seedperf) fournissent déjà la valeur : le trigger ne se déclenche pas pour elles.
CREATE TRIGGER incident_villains_fill_created_at
    BEFORE INSERT ON incident_villains
    FOR EACH ROW WHEN (NEW.incident_created_at IS NULL)
    EXECUTE FUNCTION incident_villains_fill_created_at();

-- Date de résolution d'un héros : la fin de l'intervention réussie à laquelle il a participé (confirmé).
-- NULL tant que l'intervention n'a pas réussi, ou si le héros n'est qu'invité.
ALTER TABLE intervention_heroes ADD COLUMN resolved_at timestamptz;

UPDATE intervention_heroes
   SET resolved_at = coalesce(interventions.ended_at, interventions.started_at, now())
  FROM interventions
 WHERE interventions.id = intervention_heroes.intervention_id
   AND interventions.status = 'succeeded'
   AND intervention_heroes.participation = 'confirmed';

-- Un héros confirmé sur une intervention déjà réussie (ou qui confirme son invitation) reçoit la date.
CREATE FUNCTION intervention_heroes_fill_resolved_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    SELECT CASE WHEN status = 'succeeded' THEN coalesce(ended_at, started_at, now()) END
      INTO NEW.resolved_at
      FROM interventions
     WHERE id = NEW.intervention_id;
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
       SET resolved_at = CASE WHEN n.status = 'succeeded' THEN coalesce(n.ended_at, n.started_at, now()) END
      FROM new_rows n
      JOIN old_rows o ON o.id = n.id
     WHERE intervention_heroes.intervention_id = n.id
       AND intervention_heroes.participation = 'confirmed'
       AND (n.status IS DISTINCT FROM o.status OR n.ended_at IS DISTINCT FROM o.ended_at)
       AND intervention_heroes.resolved_at IS DISTINCT FROM
           CASE WHEN n.status = 'succeeded' THEN coalesce(n.ended_at, n.started_at, now()) END;
    RETURN NULL;
END $$;

CREATE TRIGGER interventions_sync_resolved_at
    AFTER UPDATE ON interventions
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION interventions_sync_resolved_at();
