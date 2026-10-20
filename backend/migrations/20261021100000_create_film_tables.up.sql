CREATE TABLE films (
    id            uuid PRIMARY KEY,
    imdb_id       text UNIQUE,
    video_url     text,
    title         text        NOT NULL,
    release_year  smallint,
    poster_url    text,
    plot          text        NOT NULL DEFAULT '',
    raw           jsonb,
    fetched_at    timestamptz,
    created_at    timestamptz NOT NULL,
    CONSTRAINT films_source_check CHECK (imdb_id IS NOT NULL OR video_url IS NOT NULL)
);

CREATE TABLE hero_films (
    hero_id      uuid        NOT NULL REFERENCES hero_profiles (user_id) ON DELETE CASCADE,
    film_id      uuid        NOT NULL REFERENCES films (id) ON DELETE CASCADE,
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by  uuid                 REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at  timestamptz,
    created_at   timestamptz NOT NULL,
    PRIMARY KEY (hero_id, film_id)
);

CREATE INDEX hero_films_status_created_idx ON hero_films (status, created_at DESC);
CREATE INDEX hero_films_film_id_idx ON hero_films (film_id);

CREATE TABLE villain_films (
    villain_id  uuid        NOT NULL REFERENCES villains (id) ON DELETE CASCADE,
    film_id     uuid        NOT NULL REFERENCES films (id) ON DELETE CASCADE,
    created_by  uuid                 REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL,
    PRIMARY KEY (villain_id, film_id)
);

CREATE INDEX villain_films_film_id_idx ON villain_films (film_id);