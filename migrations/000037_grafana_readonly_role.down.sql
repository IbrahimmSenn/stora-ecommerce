DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    REVOKE SELECT (id, role, created_at) ON users FROM grafana_ro;
    REVOKE SELECT ON orders, order_items, products, categories, brands,
      reviews, carts, cart_items, delivery_options FROM grafana_ro;
    REVOKE USAGE ON SCHEMA public FROM grafana_ro;
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM grafana_ro', current_database());
    DROP ROLE grafana_ro;
  END IF;
END
$$;
