UPDATE products SET weight_g = 0 WHERE weight_g IS NULL;
ALTER TABLE products ALTER COLUMN weight_g SET NOT NULL;
