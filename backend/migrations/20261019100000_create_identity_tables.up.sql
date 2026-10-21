-- Comptes, sessions, jetons à usage unique et journal d'audit (voir backend/TABLES.md).
-- Les colonnes bytea marquées « chiffré » contiennent des valeurs produites par internal/crypto.

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id                 uuid PRIMARY KEY,
    email              citext      NOT NULL,
    password_hash      text        NOT NULL,
    role               text        NOT NULL DEFAULT 'citizen',
    status             text        NOT NULL DEFAULT 'pending_activation',
    civility           text        NOT NULL,
    last_name          bytea       NOT NULL, -- chiffré
    first_name         bytea       NOT NULL, -- chiffré
    birth_date         bytea       NOT NULL, -- chiffré
    phone              bytea,                -- chiffré
    street             bytea,                -- chiffré
    address_complement bytea,                -- chiffré
    postal_code        bytea,                -- chiffré
    city               bytea,                -- chiffré
    country            bytea,                -- chiffré
    avatar_path        text,
    activated_at       timestamptz,
    verified_at        timestamptz,
    verified_by        uuid,
    deactivated_at     timestamptz,
    failed_login_count smallint    NOT NULL DEFAULT 0,
    locked_until       timestamptz,
    last_login_at      timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_role_check CHECK (role IN ('citizen', 'hero', 'moderator')),
    CONSTRAINT users_status_check CHECK (status IN ('pending_activation', 'active', 'deactivated')),
    CONSTRAINT users_civility_check CHECK (civility IN ('madame', 'monsieur', 'non-precise')),
    CONSTRAINT users_failed_login_count_check CHECK (failed_login_count >= 0),
    CONSTRAINT users_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX users_role_status_idx ON users (role, status);

CREATE TABLE hero_profiles (
    user_id             uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    alias               citext      NOT NULL,
    bio                 text        NOT NULL DEFAULT '',
    request_message     text        NOT NULL DEFAULT '',
    verification_status text        NOT NULL DEFAULT 'pending',
    reviewed_by         uuid        REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at         timestamptz,
    rejection_reason    text,
    created_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT hero_profiles_alias_key UNIQUE (alias),
    CONSTRAINT hero_profiles_verification_status_check
        CHECK (verification_status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX hero_profiles_verification_status_idx ON hero_profiles (verification_status);

CREATE TABLE sessions (
    token_hash      bytea PRIMARY KEY, -- SHA-256 du jeton ; le jeton brut n'existe que dans le cookie
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL,
    remember_me     boolean     NOT NULL DEFAULT false,
    mfa_verified_at timestamptz,
    revoked_at      timestamptz,
    ip_address      inet,
    device_label    text        NOT NULL DEFAULT ''
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE user_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type       text        NOT NULL,
    token_hash bytea       NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_tokens_token_hash_key UNIQUE (token_hash),
    CONSTRAINT user_tokens_type_check CHECK (type IN ('activation', 'password_reset'))
);

CREATE INDEX user_tokens_user_id_type_idx ON user_tokens (user_id, type);

CREATE TABLE audit_logs (
    id          uuid PRIMARY KEY,
    actor_id    uuid        REFERENCES users (id) ON DELETE SET NULL,
    action      text        NOT NULL,
    entity_type text        NOT NULL DEFAULT '',
    entity_id   uuid,
    details     jsonb       NOT NULL DEFAULT '{}',
    ip_address  inet,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id);
