-- Types d'intervention (gérés par les modérateurs), vilains (créés par les utilisateurs) et incidents.
-- Les colonnes bytea marquées « chiffré » contiennent des valeurs produites par internal/crypto.

CREATE TABLE intervention_types (
    id          uuid PRIMARY KEY,
    label       citext      NOT NULL,
    description text        NOT NULL DEFAULT '',
    active      boolean     NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT intervention_types_label_key UNIQUE (label)
);

INSERT INTO intervention_types (id, label) VALUES
    (gen_random_uuid(), 'Vol'),
    (gen_random_uuid(), 'Cambriolage'),
    (gen_random_uuid(), 'Prise d''otages'),
    (gen_random_uuid(), 'Agression'),
    (gen_random_uuid(), 'Sabotage'),
    (gen_random_uuid(), 'Enlèvement');

CREATE TABLE villains (
    id          uuid PRIMARY KEY,
    alias       citext      NOT NULL,
    description text        NOT NULL DEFAULT '',
    threat_level text       NOT NULL DEFAULT 'faible',
    created_by  uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT villains_alias_key UNIQUE (alias),
    CONSTRAINT villains_threat_level_check CHECK (threat_level IN ('faible', 'modere', 'eleve', 'critique'))
);

CREATE TABLE incidents (
    id                   uuid PRIMARY KEY,
    reporter_id          uuid        NOT NULL REFERENCES users (id),
    reporter_role        text        NOT NULL,
    intervention_type_id uuid        NOT NULL REFERENCES intervention_types (id),
    villain_id           uuid        REFERENCES villains (id),
    title                text        NOT NULL,
    description          text        NOT NULL,
    address              bytea       NOT NULL, -- chiffré
    city                 bytea       NOT NULL, -- chiffré
    occurred_at          timestamptz,
    status               text        NOT NULL DEFAULT 'signale',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT incidents_reporter_role_check CHECK (reporter_role IN ('temoin', 'victime')),
    CONSTRAINT incidents_status_check CHECK (status IN ('signale', 'en-cours', 'resolu'))
);

CREATE INDEX incidents_reporter_created_idx ON incidents (reporter_id, created_at DESC);
CREATE INDEX incidents_villain_id_idx ON incidents (villain_id);
CREATE INDEX incidents_intervention_type_id_idx ON incidents (intervention_type_id);
