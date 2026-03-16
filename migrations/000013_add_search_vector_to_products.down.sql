DROP TRIGGER IF EXISTS trig_products_search_vector ON products;
DROP FUNCTION IF EXISTS products_search_vector_update();
DROP INDEX IF EXISTS idx_products_search;
ALTER TABLE products DROP COLUMN search_vector;
