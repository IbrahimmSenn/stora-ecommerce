-- Denormalize review aggregates onto products so the hot read paths (PLP, PDP,
-- search, recommendations) no longer LEFT JOIN reviews + GROUP BY AVG() on every
-- request. A trigger keeps the columns in sync; only 'approved' reviews count,
-- matching the public display rules.
ALTER TABLE products
  ADD COLUMN rating_avg NUMERIC(3,2) NOT NULL DEFAULT 0,
  ADD COLUMN rating_count INTEGER NOT NULL DEFAULT 0;

-- Backfill from existing approved reviews.
UPDATE products p SET
  rating_avg = COALESCE(sub.avg, 0),
  rating_count = COALESCE(sub.cnt, 0)
FROM (
  SELECT product_id, ROUND(AVG(rating), 2) AS avg, COUNT(*) AS cnt
  FROM reviews
  WHERE status = 'approved'
  GROUP BY product_id
) sub
WHERE p.id = sub.product_id;

-- Recompute a product's aggregate from its approved reviews. Handles INSERT,
-- DELETE, and UPDATE (including a review moving between products or changing
-- status/rating) by refreshing both the new and old product.
CREATE OR REPLACE FUNCTION products_rating_refresh() RETURNS trigger AS $$
BEGIN
  IF TG_OP IN ('INSERT', 'UPDATE') THEN
    UPDATE products SET
      rating_avg = COALESCE((SELECT ROUND(AVG(rating), 2) FROM reviews
                             WHERE product_id = NEW.product_id AND status = 'approved'), 0),
      rating_count = (SELECT COUNT(*) FROM reviews
                      WHERE product_id = NEW.product_id AND status = 'approved')
    WHERE id = NEW.product_id;
  END IF;
  IF TG_OP IN ('DELETE', 'UPDATE') THEN
    UPDATE products SET
      rating_avg = COALESCE((SELECT ROUND(AVG(rating), 2) FROM reviews
                             WHERE product_id = OLD.product_id AND status = 'approved'), 0),
      rating_count = (SELECT COUNT(*) FROM reviews
                      WHERE product_id = OLD.product_id AND status = 'approved')
    WHERE id = OLD.product_id;
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trig_products_rating_refresh
AFTER INSERT OR UPDATE OR DELETE ON reviews
FOR EACH ROW
EXECUTE FUNCTION products_rating_refresh();
