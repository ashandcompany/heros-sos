-- Journal des événements métier des incidents et des interventions (event sourcing limité à ces deux agrégats,
-- voir backend/stack/performance-et-synchronisation.md). Chaque action y laisse une ligne, écrite dans la MÊME
-- transaction que la mise à jour de l'état (tables incidents, interventions…) : l'une ne survit jamais sans
-- l'autre. Les tables d'état restent la source des lectures ; le journal dit comment on y est arrivé, et alimente
-- le temps réel (flux SSE) et la journalisation (ELK).
--
-- Le contenu des événements ne comporte que des identifiants et des états, jamais d'adresse ni de texte libre :
-- il part tel quel vers la journalisation.

CREATE TABLE events (
    -- Ordre global. Les écritures d'événements sont sérialisées (verrou consultatif pris en début de transaction) :
    -- l'ordre des numéros est celui des validations, et un lecteur qui a vu le numéro N a vu tous les précédents.
    seq            bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id       uuid        NOT NULL UNIQUE,
    aggregate_type text        NOT NULL,
    aggregate_id   uuid        NOT NULL,
    -- Rang de l'événement dans son agrégat (1, 2, 3…). L'unicité (aggregate_id, version) est la concurrence
    -- optimiste : deux écritures qui croiraient lire la même version ne peuvent pas toutes deux réussir.
    version        integer     NOT NULL,
    -- L'incident concerné (l'agrégat lui-même pour un incident) : filtre des flux et des historiques.
    incident_id    uuid        NOT NULL,
    type           text        NOT NULL,
    payload        jsonb       NOT NULL DEFAULT '{}',
    actor_id       uuid,
    -- Commande à l'origine de l'événement (clé d'idempotence), et horloge du client qui l'a créée : le délai
    -- entre les deux dates est le temps de synchronisation d'une action faite hors ligne.
    command_id     uuid,
    occurred_at    timestamptz,
    recorded_at    timestamptz NOT NULL,

    CONSTRAINT events_aggregate_version_key UNIQUE (aggregate_id, version),
    CONSTRAINT events_aggregate_type_check CHECK (aggregate_type IN ('incident', 'intervention')),
    CONSTRAINT events_version_check CHECK (version >= 1)
);

CREATE INDEX events_incident_idx ON events (incident_id, seq);
CREATE INDEX events_command_idx ON events (command_id) WHERE command_id IS NOT NULL;

-- Un journal ne se réécrit pas : toute modification ou suppression ligne à ligne est refusée.
CREATE FUNCTION events_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'events : le journal est en ajout seul (% interdit)', TG_OP USING ERRCODE = 'restrict_violation';
END $$;

CREATE TRIGGER events_append_only
    BEFORE UPDATE OR DELETE ON events
    FOR EACH ROW EXECUTE FUNCTION events_append_only();

-- Avancement de la journalisation : le dernier événement déjà écrit dans les logs, pour qu'un redémarrage ne
-- perde ni ne répète rien.
CREATE TABLE event_publication (
    name     text   PRIMARY KEY,
    last_seq bigint NOT NULL DEFAULT 0
);
