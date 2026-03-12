CREATE TABLE reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  comment TEXT CHECK (char_length(comment) <= 2000),
  rating smallint NOT NULL CHECK(rating between 1 and 5),
  UNIQUE (user_id, product_id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TRIGGER update_reviews_modtime
BEFORE UPDATE ON reviews
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();