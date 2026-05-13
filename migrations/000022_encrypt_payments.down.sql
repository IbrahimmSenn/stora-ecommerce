DROP INDEX IF EXISTS idx_payments_pi_hmac;

ALTER TABLE payments
  ADD COLUMN stripe_payment_intent_id TEXT,
  ADD COLUMN stripe_refund_id TEXT,
  ADD COLUMN error_code TEXT,
  ADD COLUMN error_message TEXT;

ALTER TABLE payments
  DROP COLUMN stripe_payment_intent_id_enc,
  DROP COLUMN stripe_payment_intent_id_hmac,
  DROP COLUMN stripe_refund_id_enc,
  DROP COLUMN error_code_enc,
  DROP COLUMN error_message_enc;

-- The original migration declared a UNIQUE constraint inline. Re-add it.
ALTER TABLE payments
  ADD CONSTRAINT payments_stripe_payment_intent_id_key UNIQUE (stripe_payment_intent_id);

CREATE INDEX IF NOT EXISTS idx_payments_intent_id
  ON payments(stripe_payment_intent_id);
