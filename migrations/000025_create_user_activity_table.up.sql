CREATE TABLE IF NOT EXISTS user_activity (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    guest_session_id UUID,
    event_type      TEXT NOT NULL CHECK (event_type IN ('view', 'search', 'add_to_cart', 'purchase')),
    product_id      UUID REFERENCES products(id) ON DELETE CASCADE,
    category_id     UUID REFERENCES categories(id) ON DELETE CASCADE,
    search_query    TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (user_id IS NOT NULL OR guest_session_id IS NOT NULL),
    CHECK (
        (event_type = 'search' AND search_query IS NOT NULL)
        OR (event_type <> 'search' AND product_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS user_activity_user_recent
    ON user_activity (user_id, occurred_at DESC)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS user_activity_guest_recent
    ON user_activity (guest_session_id, occurred_at DESC)
    WHERE guest_session_id IS NOT NULL;
