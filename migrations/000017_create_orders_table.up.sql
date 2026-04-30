CREATE TABLE orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_number VARCHAR(32) UNIQUE NOT NULL,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  guest_session_id UUID,
  status VARCHAR(32) NOT NULL CHECK (status IN (
    'pending_payment','paid','payment_failed','processing','shipped','delivered','cancelled','refunded'
  )),
  email_encrypted BYTEA NOT NULL,
  phone_encrypted BYTEA,
  subtotal_cents INT NOT NULL CHECK (subtotal_cents >= 0),
  shipping_cents INT NOT NULL CHECK (shipping_cents >= 0),
  total_cents INT NOT NULL CHECK (total_cents >= 0),
  shipping_method VARCHAR(32) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT orders_owner_check CHECK (
    user_id IS NOT NULL OR guest_session_id IS NOT NULL
  )
);

CREATE INDEX idx_orders_user_id ON orders(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_orders_guest_session ON orders(guest_session_id) WHERE guest_session_id IS NOT NULL;
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

CREATE TRIGGER update_orders_modtime
  BEFORE UPDATE ON orders
  FOR EACH ROW
  EXECUTE PROCEDURE update_modified_column();
