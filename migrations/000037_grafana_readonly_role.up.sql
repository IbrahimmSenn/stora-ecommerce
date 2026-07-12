-- Read-only role for the Grafana Postgres datasource (business dashboards).
-- SELECT on commerce tables only; users is a column-level grant so the
-- encrypted PII (email_encrypted, email_hmac, password_hash) is unreadable
-- even if the Grafana credentials leak. Dev-only password — the monitoring
-- stack never runs against a production database in this project.
--
-- Roles are cluster-level, so CREATE is wrapped to tolerate reruns against a
-- cluster where the role already exists (e.g. after a down+up cycle).
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    CREATE ROLE grafana_ro LOGIN PASSWORD 'grafana-ro-dev-password';
  END IF;
  -- Dynamic: the database is `mystore` in compose but `ci` in the pipeline.
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO grafana_ro', current_database());
END
$$;

GRANT USAGE ON SCHEMA public TO grafana_ro;

GRANT SELECT ON orders, order_items, products, categories, brands,
  reviews, carts, cart_items, delivery_options TO grafana_ro;
GRANT SELECT (id, role, created_at) ON users TO grafana_ro;
