-- Double authentification par TOTP (voir backend/tables/comptes.md, §« user_mfa et mfa_recovery_codes »).

CREATE TABLE user_mfa (
    user_id        uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    -- Secret TOTP chiffré (internal/crypto), comme les données personnelles : une fuite de la base ne
    -- permet pas de générer les codes.
    totp_secret    bytea       NOT NULL,
    -- NULL tant que l'utilisateur n'a pas prouvé, par un premier code, que son application est configurée :
    -- une activation abandonnée ne protège (ni ne bloque) personne.
    confirmed_at   timestamptz,
    -- Dernier pas de 30 secondes accepté : un code ne sert qu'une fois, même dans sa fenêtre de validité.
    last_used_step bigint      NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE mfa_recovery_codes (
    id        uuid PRIMARY KEY,
    user_id   uuid  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- SHA-256 du code (80 bits aléatoires : inutile de le saler) ; le code lui-même n'est montré qu'une fois.
    code_hash bytea NOT NULL,
    used_at   timestamptz,

    CONSTRAINT mfa_recovery_codes_user_hash_key UNIQUE (user_id, code_hash)
);
