-- weight_g is genuinely optional for many product types (digital goods,
-- marketplace listings, configurable bundles). Relax the NOT NULL so the
-- catalogue can hold items where weight is unknown.
ALTER TABLE products ALTER COLUMN weight_g DROP NOT NULL;
