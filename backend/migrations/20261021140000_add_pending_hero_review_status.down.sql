UPDATE hero_reviews SET status = 'published' WHERE status = 'pending';

ALTER TABLE hero_reviews
    DROP CONSTRAINT hero_reviews_status_check;

ALTER TABLE hero_reviews
    ADD CONSTRAINT hero_reviews_status_check CHECK (status IN ('published', 'hidden'));
