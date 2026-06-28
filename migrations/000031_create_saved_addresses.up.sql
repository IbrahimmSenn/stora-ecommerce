-- User-owned saved addresses for faster checkout. All address fields are
-- AES-256-GCM encrypted at rest (same as order shipping addresses); only the
-- non-sensitive label and is_default flag are stored in clear.
CREATE TABLE saved_addresses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  label VARCHAR(60),
  recipient_name_encrypted BYTEA NOT NULL,
  line1_encrypted BYTEA NOT NULL,
  line2_encrypted BYTEA,
  city_encrypted BYTEA NOT NULL,
  region_encrypted BYTEA NOT NULL,
  postal_code_encrypted BYTEA NOT NULL,
  country_encrypted BYTEA NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_saved_addresses_user ON saved_addresses(user_id);

CREATE TRIGGER update_saved_addresses_modtime
  BEFORE UPDATE ON saved_addresses
  FOR EACH ROW
  EXECUTE PROCEDURE update_modified_column();
