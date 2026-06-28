DROP TABLE IF EXISTS review_votes;
DROP INDEX IF EXISTS idx_reviews_status;
DROP INDEX IF EXISTS idx_reviews_product_status;
ALTER TABLE reviews DROP COLUMN IF EXISTS status;
