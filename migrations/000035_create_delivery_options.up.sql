-- Admin-managed delivery (shipping) options, replacing the hardcoded standard/
-- express constants. `code` is the stable identifier stored on orders
-- (orders.shipping_method); the two seeded rows match the previous constants so
-- existing orders and the checkout flow are unchanged.
CREATE TABLE delivery_options (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        VARCHAR(40) NOT NULL UNIQUE,
  label       VARCHAR(100) NOT NULL,
  price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
  eta_label   VARCHAR(100) NOT NULL DEFAULT '',
  sort_order  INTEGER NOT NULL DEFAULT 0,
  active      BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO delivery_options (code, label, price_cents, eta_label, sort_order) VALUES
  ('standard', 'Standard', 500,  '5–7 business days', 1),
  ('express',  'Express',  1500, '1–2 business days', 2);
