CREATE TABLE shipping_addresses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
  recipient_name_encrypted BYTEA NOT NULL,
  line1_encrypted BYTEA NOT NULL,
  line2_encrypted BYTEA,
  city_encrypted BYTEA NOT NULL,
  region_encrypted BYTEA NOT NULL,
  postal_code_encrypted BYTEA NOT NULL,
  country_encrypted BYTEA NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
