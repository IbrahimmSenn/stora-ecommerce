-- Dedupe any (product_id, url) duplicates that snuck in via re-running the
-- seed before the seed itself was idempotent. Keep the earliest row per pair.
DELETE FROM product_images a
USING product_images b
WHERE a.id > b.id
  AND a.product_id = b.product_id
  AND a.url = b.url;

ALTER TABLE product_images
  ADD CONSTRAINT product_images_product_url_unique
  UNIQUE (product_id, url);
