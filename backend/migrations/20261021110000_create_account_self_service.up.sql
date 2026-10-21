-- Libre-service du compte : changement d'adresse email soumis à un modérateur, et abonnements à la
-- newsletter (voir backend/TABLES.md, §« email_change_requests » et §« newsletter_subscriptions »).

CREATE TABLE email_change_requests (
    id          uuid PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    new_email   citext      NOT NULL,
    status      text        NOT NULL DEFAULT 'pending',
    reviewed_by uuid        REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT email_change_requests_status_check CHECK (status IN ('pending', 'approved', 'rejected'))
);

-- Un compte n'a jamais plus d'une demande en attente : une nouvelle demande remplace la précédente.
CREATE UNIQUE INDEX email_change_requests_one_pending_idx ON email_change_requests (user_id) WHERE status = 'pending';
-- File de travail des modérateurs : les demandes d'un statut, les plus anciennes d'abord.
CREATE INDEX email_change_requests_status_created_at_idx ON email_change_requests (status, created_at);

CREATE TABLE newsletter_subscriptions (
    id              uuid PRIMARY KEY,
    email           citext      NOT NULL,
    user_id         uuid        REFERENCES users (id) ON DELETE SET NULL,
    status          text        NOT NULL DEFAULT 'pending',
    source          text        NOT NULL,
    confirmed_at    timestamptz, -- preuve de consentement : date de la confirmation
    unsubscribed_at timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT newsletter_subscriptions_email_key UNIQUE (email),
    CONSTRAINT newsletter_subscriptions_status_check CHECK (status IN ('pending', 'subscribed', 'unsubscribed')),
    CONSTRAINT newsletter_subscriptions_source_check CHECK (source IN ('footer', 'account')),
    -- Pas d'abonné sans preuve de consentement.
    CONSTRAINT newsletter_subscriptions_consent_check CHECK (status <> 'subscribed' OR confirmed_at IS NOT NULL)
);

CREATE INDEX newsletter_subscriptions_user_id_idx ON newsletter_subscriptions (user_id);
-- Purge des demandes jamais confirmées.
CREATE INDEX newsletter_subscriptions_status_created_at_idx ON newsletter_subscriptions (status, created_at);
