-- Moderation status for reviews: new reviews land in 'pending' for the admin
-- queue; only 'approved' reviews are shown publicly and counted in averages.
ALTER TABLE reviews
  ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'pending'
  CHECK (status IN ('pending', 'approved', 'hidden'));

-- Any rows that predate this column are treated as approved so historical
-- ratings are preserved.
UPDATE reviews SET status = 'approved';

CREATE INDEX idx_reviews_product_status ON reviews(product_id, status);
CREATE INDEX idx_reviews_status ON reviews(status);

-- "Helpful" votes. One vote per user per review.
CREATE TABLE review_votes (
  review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (review_id, user_id)
);

CREATE INDEX idx_review_votes_review ON review_votes(review_id);
