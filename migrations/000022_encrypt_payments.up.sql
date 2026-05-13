-- Encrypt sensitive payment fields at rest. Stripe identifier columns become
-- AES-256-GCM bytea ciphertext, paired with an HMAC-SHA256 lookup column for
-- equality search (webhook arrives with the plaintext intent id and needs to
-- find the row). Error code + message also encrypted — Stripe error messages
-- can carry free-form text and shouldn't be readable from a DB dump.

ALTER TABLE payments
  ADD COLUMN stripe_payment_intent_id_enc BYTEA,
  ADD COLUMN stripe_payment_intent_id_hmac BYTEA,
  ADD COLUMN stripe_refund_id_enc BYTEA,
  ADD COLUMN error_code_enc BYTEA,
  ADD COLUMN error_message_enc BYTEA;

-- The auto-generated UNIQUE constraint on stripe_payment_intent_id created a
-- backing index of the same name; dropping the column drops the index too.
ALTER TABLE payments
  DROP COLUMN stripe_payment_intent_id,
  DROP COLUMN stripe_refund_id,
  DROP COLUMN error_code,
  DROP COLUMN error_message;

CREATE UNIQUE INDEX idx_payments_pi_hmac
  ON payments(stripe_payment_intent_id_hmac)
  WHERE stripe_payment_intent_id_hmac IS NOT NULL;
