-- Funnel panels (view -> add_to_cart -> purchase) read user_activity from
-- Grafana. Column-level grant, same PII stance as 000037: search_query is
-- free-text user input and stays unreadable to the dashboard role.
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    GRANT SELECT (id, user_id, guest_session_id, event_type, product_id, category_id, occurred_at)
      ON user_activity TO grafana_ro;
  END IF;
END
$$;
