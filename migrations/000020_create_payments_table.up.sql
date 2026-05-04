CREATE TABLE payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  stripe_payment_intent_id TEXT NOT NULL UNIQUE,
  status VARCHAR(32) NOT NULL CHECK (status IN ('pending','succeeded','failed','cancelled')),
  amount_cents INT NOT NULL CHECK (amount_cents >= 0),
  currency VARCHAR(8) NOT NULL DEFAULT 'usd',
  error_code TEXT,
  error_message TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_intent_id ON payments(stripe_payment_intent_id);

CREATE TRIGGER update_payments_modtime
  BEFORE UPDATE ON payments
  FOR EACH ROW
  EXECUTE PROCEDURE update_modified_column();
