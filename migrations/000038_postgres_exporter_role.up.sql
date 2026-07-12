-- Monitoring role for postgres_exporter (Prometheus). Separate from
-- grafana_ro on purpose: pg_monitor exposes pg_stat_activity query text,
-- which can contain literal parameter values — different credential,
-- different blast radius, and grafana_ro keeps its minimal table grants.
-- Dev-only password, same reasoning as 000037.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'postgres_exporter') THEN
    CREATE ROLE postgres_exporter LOGIN PASSWORD 'postgres-exporter-dev-password';
  END IF;
  -- Dynamic: the database is `mystore` in compose but `ci` in the pipeline.
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO postgres_exporter', current_database());
END
$$;

GRANT pg_monitor TO postgres_exporter;
