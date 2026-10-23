-- Newsletters rédigées par les modérateurs (voir backend/tables/contenu.md, §« newsletters »).
-- Brouillon : sent_at IS NULL. Envoyée : sent_at renseigné, le message n'est plus modifiable.

CREATE TABLE newsletters (
    id                uuid PRIMARY KEY,
    subject           text        NOT NULL,
    body_html         text        NOT NULL,
    -- NULL = tous les abonnés ; renseigné = un seul utilisateur. Pas de SET NULL : la suppression d'un
    -- compte ne doit jamais transformer un message ciblé en message envoyé à tous.
    recipient_user_id uuid        REFERENCES users (id) ON DELETE CASCADE,
    created_by        uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL,
    sent_at           timestamptz
);

CREATE INDEX newsletters_created_at_idx ON newsletters (created_at DESC, id DESC);
