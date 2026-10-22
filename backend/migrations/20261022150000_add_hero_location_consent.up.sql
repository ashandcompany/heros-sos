-- Consentement explicite au partage de position (voir backend/tables/comptes.md, §« hero_locations »).
-- NULL = pas de consentement. Retirer le consentement remet la colonne à NULL et supprime la position.

ALTER TABLE hero_profiles ADD COLUMN location_consent_at timestamptz;
