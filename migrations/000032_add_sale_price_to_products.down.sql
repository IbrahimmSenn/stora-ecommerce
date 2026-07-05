ALTER TABLE products DROP CONSTRAINT IF EXISTS products_sale_price_below_price;
ALTER TABLE products DROP COLUMN IF EXISTS sale_price;
