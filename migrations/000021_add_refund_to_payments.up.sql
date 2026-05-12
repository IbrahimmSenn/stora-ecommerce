ALTER TABLE payments
  DROP CONSTRAINT payments_status_check;

ALTER TABLE payments
  ADD CONSTRAINT payments_status_check
  CHECK (status IN ('pending','succeeded','failed','cancelled','refunded'));

ALTER TABLE payments
  ADD COLUMN stripe_refund_id TEXT,
  ADD COLUMN refunded_at TIMESTAMP WITH TIME ZONE;
