-- Titre court de l'incident, affiché dans les listes. Les incidents déjà déclarés reçoivent un titre
-- vide : le frontend affiche alors le type et la ville.
ALTER TABLE incidents
    ADD COLUMN title text NOT NULL DEFAULT '';
