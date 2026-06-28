DROP TRIGGER IF EXISTS trig_products_rating_refresh ON reviews;
DROP FUNCTION IF EXISTS products_rating_refresh();
ALTER TABLE products
  DROP COLUMN IF EXISTS rating_avg,
  DROP COLUMN IF EXISTS rating_count;
