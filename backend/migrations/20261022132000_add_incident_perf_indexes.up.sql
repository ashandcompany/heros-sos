-- Index de lecture des listes d'incidents et des historiques publics, ajoutés après mesure sur 5 millions
-- d'incidents (voir backend/stack/performance-et-synchronisation.md). Chacun remplace un parcours complet de table
-- ou un tri de plusieurs milliers de lignes par la lecture des 20 premières entrées de l'index.
--
-- Cette migration ne contient QUE des index, sans toucher aux données : `make perf-indexes-off` les retire (et
-- `make perf-indexes-on` les recrée), ce qui rend la lenteur d'origine reproductible pour une démonstration
-- avant/après. Sur une base de production déjà volumineuse, CREATE INDEX bloque les écritures le temps de sa
-- construction : à lancer en période creuse.

-- Liste sans filtre : les plus récents d'abord (le tri de GET /incidents).
CREATE INDEX incidents_created_idx ON incidents (created_at DESC, id DESC);

-- Liste filtrée par statut : l'index existant (status, priority, created_at) ne peut pas servir ce tri, car la
-- priorité s'intercale entre le statut et la date.
CREATE INDEX incidents_status_created_idx ON incidents (status, created_at DESC, id DESC);

-- « Mes incidents », fiches de réputation : aucun index n'existait sur reporter_id (une clé étrangère n'en crée
-- pas dans PostgreSQL), donc la suppression d'un compte parcourait elle aussi toute la table.
CREATE INDEX incidents_reporter_created_idx ON incidents (reporter_id, created_at DESC, id DESC);

-- Historique d'un vilain : ses incidents, du plus récent au plus ancien, sans rien trier.
CREATE INDEX incident_villains_villain_idx ON incident_villains (villain_id, incident_created_at DESC, incident_id DESC);

-- Dernière intervention d'un incident (jointure latérale de l'historique des vilains) et recherche par incident.
CREATE INDEX interventions_incident_idx ON interventions (incident_id, started_at DESC NULLS LAST, id DESC);

-- Historique d'un héros : les incidents qu'il a résolus, du plus récent au plus ancien.
CREATE INDEX intervention_heroes_resolved_idx ON intervention_heroes (hero_id, resolved_at DESC, intervention_id DESC)
    WHERE resolved_at IS NOT NULL;
