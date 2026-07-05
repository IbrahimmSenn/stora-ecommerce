-- Multi-size image variants. Existing rows keep only `url`; uploads through the
-- new pipeline populate all three. Readers COALESCE down to `url` for back-compat.
ALTER TABLE product_images
  ADD COLUMN thumbnail_url TEXT,
  ADD COLUMN card_url      TEXT,
  ADD COLUMN full_url      TEXT;
