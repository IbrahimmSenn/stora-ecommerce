ALTER TABLE product_images
  DROP COLUMN IF EXISTS thumbnail_url,
  DROP COLUMN IF EXISTS card_url,
  DROP COLUMN IF EXISTS full_url;
