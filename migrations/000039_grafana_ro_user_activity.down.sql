DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    REVOKE SELECT (id, user_id, guest_session_id, event_type, product_id, category_id, occurred_at)
      ON user_activity FROM grafana_ro;
  END IF;
END
$$;
