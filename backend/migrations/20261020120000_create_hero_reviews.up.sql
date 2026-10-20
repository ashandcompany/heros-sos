CREATE TABLE hero_reviews (
    id              uuid PRIMARY KEY,
    intervention_id uuid        NOT NULL,
    hero_id         uuid        NOT NULL,
    citizen_id      uuid        REFERENCES users (id) ON DELETE SET NULL,
    rating          smallint    NOT NULL,
    comment         text,
    status          text        NOT NULL DEFAULT 'published',
    moderated_by    uuid        REFERENCES users (id) ON DELETE SET NULL,
    moderated_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT hero_reviews_intervention_hero_fkey
        FOREIGN KEY (intervention_id, hero_id)
        REFERENCES intervention_heroes (intervention_id, hero_id),
    CONSTRAINT hero_reviews_intervention_hero_key UNIQUE (intervention_id, hero_id),
    CONSTRAINT hero_reviews_rating_check CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT hero_reviews_status_check CHECK (status IN ('published', 'hidden'))
);

CREATE INDEX hero_reviews_hero_status_created_idx
    ON hero_reviews (hero_id, status, created_at DESC);
CREATE INDEX hero_reviews_status_created_idx
    ON hero_reviews (status, created_at DESC);

CREATE TABLE hero_review_replies (
    review_id    uuid        PRIMARY KEY REFERENCES hero_reviews (id) ON DELETE CASCADE,
    hero_id      uuid        NOT NULL REFERENCES hero_profiles (user_id),
    content      text        NOT NULL,
    status       text        NOT NULL DEFAULT 'published',
    moderated_by uuid        REFERENCES users (id) ON DELETE SET NULL,
    moderated_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    edited_at    timestamptz,

    CONSTRAINT hero_review_replies_status_check CHECK (status IN ('published', 'pending', 'hidden'))
);

CREATE INDEX hero_review_replies_status_created_idx
    ON hero_review_replies (status, created_at DESC);