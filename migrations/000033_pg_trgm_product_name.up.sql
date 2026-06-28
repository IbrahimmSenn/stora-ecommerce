-- Trigram index for the autocomplete/suggest query, which matches product
-- names with a leading-wildcard ILIKE '%term%'. A btree index can't serve that;
-- a GIN trigram index can, keeping suggestions fast as the catalog grows.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_products_name_trgm ON products USING gin (name gin_trgm_ops);
