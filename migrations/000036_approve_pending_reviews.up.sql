-- Reviews are now published immediately (auto-approved on submission), with
-- admins able to hide or delete inappropriate ones afterward. Flip any reviews
-- still sitting in the old 'pending' state to 'approved' so they go live. The
-- trig_products_rating_refresh trigger recomputes each product's rating_avg /
-- rating_count on these updates.
UPDATE reviews SET status = 'approved' WHERE status = 'pending';
