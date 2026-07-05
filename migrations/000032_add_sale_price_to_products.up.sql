-- sale_price holds a discounted price (cents) when a product is on sale.
-- NULL means no active discount. When set it must be below the regular price.
ALTER TABLE products ADD COLUMN sale_price BIGINT;
ALTER TABLE products ADD CONSTRAINT products_sale_price_below_price
    CHECK (sale_price IS NULL OR sale_price < price);
