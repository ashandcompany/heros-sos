-- Position actuelle des héros (voir backend/tables/comptes.md, §« hero_locations »).
-- En clair (nécessaire aux calculs de distance), une seule ligne par héros, sans historique.

CREATE TABLE hero_locations (
    hero_id    uuid PRIMARY KEY REFERENCES hero_profiles (user_id) ON DELETE CASCADE,
    latitude   numeric(9, 6) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude  numeric(9, 6) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    accuracy_m integer CHECK (accuracy_m >= 0),
    updated_at timestamptz NOT NULL
);

-- Nettoyage des positions périmées.
CREATE INDEX hero_locations_updated_at_idx ON hero_locations (updated_at);
