-- La nouvelle adresse d'un compte doit être confirmée par son titulaire (lien envoyé à cette adresse)
-- avant qu'un modérateur puisse accepter le changement : le modérateur valide une demande légitime, le
-- lien prouve que l'adresse existe et appartient bien à la personne qui la demande.
ALTER TABLE email_change_requests ADD COLUMN confirmed_at timestamptz;
