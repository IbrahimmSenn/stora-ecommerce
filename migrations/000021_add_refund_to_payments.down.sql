ALTER TABLE payments
  DROP COLUMN IF EXISTS refunded_at,
  DROP COLUMN IF EXISTS stripe_refund_id;

ALTER TABLE payments
  DROP CONSTRAINT payments_status_check;

ALTER TABLE payments
  ADD CONSTRAINT payments_status_check
  CHECK (status IN ('pending','succeeded','failed','cancelled'));
