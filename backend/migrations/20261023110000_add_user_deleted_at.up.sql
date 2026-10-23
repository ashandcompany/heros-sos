-- Droit à l'effacement : un compte supprimé est anonymisé, pas effacé, pour que ses incidents, ses
-- interventions et ses avis restent cohérents. deleted_at date l'anonymisation ; NULL = compte en service.
-- Un compte anonymisé reste « deactivated » et ne peut plus être réactivé.

ALTER TABLE users ADD COLUMN deleted_at timestamptz;
