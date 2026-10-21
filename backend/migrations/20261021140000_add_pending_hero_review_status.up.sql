ALTER TABLE hero_reviews
    DROP CONSTRAINT hero_reviews_status_check;

ALTER TABLE hero_reviews
    ADD CONSTRAINT hero_reviews_status_check CHECK (status IN ('published', 'pending', 'hidden'));
