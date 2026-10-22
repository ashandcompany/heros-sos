-- Les derniers incidents résolus du public et des héros (incidents approuvés) : lus dans l'ordre de l'index, sans tri
-- (GET /incidents?status=succeeded&sort=-resolvedAt, la page d'accueil). Index partiel : il ne sert que si la requête
-- écrit `moderation = 'approved'` en toutes lettres.
--
-- Comme add_incident_perf_indexes, cette migration ne contient QUE des index : `make perf-indexes-off` la retire (et
-- `make perf-indexes-on` la recrée) avec les autres. CREATE INDEX bloque les écritures pendant sa construction : à
-- lancer en période creuse sur une base volumineuse.
CREATE INDEX incidents_approved_resolved_idx ON incidents (resolved_at DESC NULLS LAST, created_at DESC, id DESC)
    WHERE moderation = 'approved';
