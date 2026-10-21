CREATE TABLE articles (
    id            uuid PRIMARY KEY,
    title         text        NOT NULL,
    slug          text        NOT NULL,
    excerpt       text        NOT NULL DEFAULT '',
    content       text        NOT NULL,
    status        text        NOT NULL DEFAULT 'brouillon' CHECK (status IN ('brouillon', 'publie')),
    author_id     uuid                 REFERENCES users (id) ON DELETE SET NULL,
    author_name   text        NOT NULL,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL,
    published_at  timestamptz,
    CONSTRAINT articles_slug_key UNIQUE (slug),
    CONSTRAINT articles_published_check CHECK (status = 'brouillon' OR published_at IS NOT NULL)
);

CREATE INDEX articles_status_published_idx ON articles (status, published_at DESC);
CREATE INDEX articles_updated_at_idx ON articles (updated_at DESC);
