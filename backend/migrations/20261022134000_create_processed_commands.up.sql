-- Commandes déjà traitées : la clé d'idempotence d'une requête (en-tête Idempotency-Key) et la réponse qu'elle a
-- reçue. Une action faite hors ligne est rejouée au retour du réseau, parfois plusieurs fois (connexion coupée
-- pendant l'envoi) : la clé garantit qu'elle ne s'applique qu'une fois, et qu'un rejeu reçoit la même réponse
-- (voir backend/stack/performance-et-synchronisation.md).
--
-- La clé est propre à son auteur : un utilisateur ne peut ni rejouer, ni deviner, ni bloquer la clé d'un autre.

CREATE TABLE processed_commands (
    user_id      uuid        NOT NULL,
    command_id   uuid        NOT NULL,
    method       text        NOT NULL,
    path         text        NOT NULL,
    -- Empreinte de la requête (méthode, chemin, corps) : la même clé avec une autre requête est une erreur du client.
    request_hash text        NOT NULL,
    -- Réponse rendue au premier passage (seules les réussites sont conservées : un échec annule tout et se rejoue).
    status_code  smallint    NOT NULL DEFAULT 0,
    content_type text        NOT NULL DEFAULT '',
    response     bytea       NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL,

    CONSTRAINT processed_commands_pk PRIMARY KEY (user_id, command_id)
);

-- Purge périodique des commandes trop anciennes pour être rejouées.
CREATE INDEX processed_commands_created_idx ON processed_commands (created_at);
