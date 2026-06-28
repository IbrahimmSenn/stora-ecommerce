-- Constrain user roles to the supported RBAC set.
ALTER TABLE users
  ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'support', 'sales', 'customer'));

-- Audit trail for privileged (staff) actions. Stores the actor by id + role
-- only (no email snapshot) so it stays consistent with PII-at-rest encryption;
-- the current email is resolved by joining users when the log is viewed.
CREATE TABLE admin_audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
  actor_role VARCHAR(16),
  action VARCHAR(8) NOT NULL,         -- HTTP method
  target TEXT NOT NULL,               -- request path
  status_code INT NOT NULL,
  ip TEXT,
  occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_actor ON admin_audit_log(actor_id);
CREATE INDEX idx_admin_audit_occurred ON admin_audit_log(occurred_at DESC);
