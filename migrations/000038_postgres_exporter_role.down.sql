DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'postgres_exporter') THEN
    REVOKE pg_monitor FROM postgres_exporter;
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM postgres_exporter', current_database());
    DROP ROLE postgres_exporter;
  END IF;
END
$$;
