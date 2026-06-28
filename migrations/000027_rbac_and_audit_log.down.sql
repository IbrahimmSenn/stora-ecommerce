DROP TABLE IF EXISTS admin_audit_log;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
