-- Deux index de lecture ajoutés après mesure (voir backend/tables/comptes.md, §« sessions », et
-- backend/tables/incidents.md, §« intervention_heroes »).

-- Les listes de héros agrègent les interventions confirmées de quelques héros seulement : sans index sur
-- hero_id, la clé primaire (intervention_id, hero_id) obligeait à lire toute la table.
CREATE INDEX intervention_heroes_hero_id_idx ON intervention_heroes (hero_id);

-- Le nettoyage périodique supprime les sessions expirées OU révoquées : seul expires_at était indexé, donc la
-- condition OR parcourait toute la table à chaque passage. Index partiel : seules les sessions révoquées y sont.
CREATE INDEX sessions_revoked_at_idx ON sessions (revoked_at) WHERE revoked_at IS NOT NULL;
