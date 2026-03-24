-- Seed data for testing / demo purposes
-- Admin user: admin@shop.com / admin123
-- Customer user: customer@shop.com / customer123

-- Admin user (password: admin123)
INSERT INTO users (id, email, password_hash, role) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'admin@shop.com',
   '$2b$10$eoPLJ/PW4skcSOuKY0lcuuegPRmubIawIrDGyByoK8yzn.sSGY9Qq', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Customer user (password: customer123)
INSERT INTO users (id, email, password_hash, role) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'customer@shop.com',
   '$2b$10$/ieJ0h2MEnPkr/TH9zhvpOVZ01GS2biGBUNZ1JMspSmBiwUOhxgxG', 'customer')
ON CONFLICT (email) DO NOTHING;

-- Categories (hierarchical)
INSERT INTO categories (id, name, slug, parent_id) VALUES
  ('ca000000-0000-0000-0000-000000000001', 'Electronics',    'electronics',    NULL),
  ('ca000000-0000-0000-0000-000000000002', 'Smartphones',    'smartphones',    'ca000000-0000-0000-0000-000000000001'),
  ('ca000000-0000-0000-0000-000000000003', 'Laptops',        'laptops',        'ca000000-0000-0000-0000-000000000001'),
  ('ca000000-0000-0000-0000-000000000004', 'Clothing',       'clothing',       NULL),
  ('ca000000-0000-0000-0000-000000000005', 'Men',            'men',            'ca000000-0000-0000-0000-000000000004'),
  ('ca000000-0000-0000-0000-000000000006', 'Women',          'women',          'ca000000-0000-0000-0000-000000000004'),
  ('ca000000-0000-0000-0000-000000000007', 'Home & Kitchen', 'home-kitchen',   NULL)
ON CONFLICT (name) DO NOTHING;

-- Brands
INSERT INTO brands (id, name) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'Apple'),
  ('b0000000-0000-0000-0000-000000000002', 'Samsung'),
  ('b0000000-0000-0000-0000-000000000003', 'Nike'),
  ('b0000000-0000-0000-0000-000000000004', 'Sony'),
  ('b0000000-0000-0000-0000-000000000005', 'IKEA')
ON CONFLICT (name) DO NOTHING;

-- Products
INSERT INTO products (id, name, description, price, stock_quantity, category_id, brand_id, weight_g, dimensions_cm) VALUES
  ('a0000000-0000-0000-0000-000000000001',
   'iPhone 15 Pro', 'Latest Apple smartphone with A17 Pro chip, titanium design, and 48MP camera system.',
   99900, 50, 'ca000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000001', 187, 14.69),

  ('a0000000-0000-0000-0000-000000000002',
   'Samsung Galaxy S24 Ultra', 'Premium Android phone with S Pen, 200MP camera, and titanium frame.',
   119900, 35, 'ca000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 232, 16.28),

  ('a0000000-0000-0000-0000-000000000003',
   'MacBook Air M3', '13-inch laptop with Apple M3 chip, 18-hour battery, and Liquid Retina display.',
   109900, 25, 'ca000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000001', 1240, 30.41),

  ('a0000000-0000-0000-0000-000000000004',
   'Samsung Galaxy Book4 Pro', '14-inch AMOLED laptop with Intel Core Ultra processor.',
   149900, 15, 'ca000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000002', 1230, 31.24),

  ('a0000000-0000-0000-0000-000000000005',
   'Nike Air Max 90', 'Classic running shoe with visible Air cushioning and iconic design.',
   12999, 100, 'ca000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000003', 340, 30.00),

  ('a0000000-0000-0000-0000-000000000006',
   'Nike Dri-FIT T-Shirt', 'Lightweight training tee with sweat-wicking technology.',
   3499, 200, 'ca000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000003', 150, NULL),

  ('a0000000-0000-0000-0000-000000000007',
   'Sony WH-1000XM5', 'Industry-leading noise canceling wireless headphones with 30-hour battery.',
   34999, 60, 'ca000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000004', 250, 20.00),

  ('a0000000-0000-0000-0000-000000000008',
   'Sony PlayStation 5', 'Next-gen gaming console with ultra-high speed SSD and ray tracing.',
   49999, 20, 'ca000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000004', 4500, 39.00),

  ('a0000000-0000-0000-0000-000000000009',
   'IKEA KALLAX Shelf', 'Versatile shelving unit, 4x2 configuration. Perfect for organizing any room.',
   6999, 40, 'ca000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000005', 25500, 147.00),

  ('a0000000-0000-0000-0000-000000000010',
   'IKEA MALM Desk', 'Work desk with pull-out panel and cable management.',
   17999, 30, 'ca000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000005', 35000, 140.00)
ON CONFLICT (id) DO NOTHING;

-- Product images
INSERT INTO product_images (product_id, url, is_primary) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'https://picsum.photos/seed/iphone15/400/400', true),
  ('a0000000-0000-0000-0000-000000000001', 'https://picsum.photos/seed/iphone15b/400/400', false),
  ('a0000000-0000-0000-0000-000000000002', 'https://picsum.photos/seed/galaxy24/400/400', true),
  ('a0000000-0000-0000-0000-000000000003', 'https://picsum.photos/seed/macbook/400/400', true),
  ('a0000000-0000-0000-0000-000000000004', 'https://picsum.photos/seed/galaxybook/400/400', true),
  ('a0000000-0000-0000-0000-000000000005', 'https://picsum.photos/seed/airmax/400/400', true),
  ('a0000000-0000-0000-0000-000000000006', 'https://picsum.photos/seed/niketee/400/400', true),
  ('a0000000-0000-0000-0000-000000000007', 'https://picsum.photos/seed/sonyxm5/400/400', true),
  ('a0000000-0000-0000-0000-000000000008', 'https://picsum.photos/seed/ps5/400/400', true),
  ('a0000000-0000-0000-0000-000000000009', 'https://picsum.photos/seed/kallax/400/400', true),
  ('a0000000-0000-0000-0000-000000000010', 'https://picsum.photos/seed/malm/400/400', true);

-- Reviews (to test rating-based search/sort)
INSERT INTO reviews (product_id, user_id, rating, comment) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 5, 'Best phone I have ever used!'),
  ('a0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 4, 'Great phone, camera is amazing.'),
  ('a0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 5, 'Incredible battery life and performance.'),
  ('a0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000001', 4, 'Very comfortable, classic style.'),
  ('a0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000001', 5, 'Noise cancellation is unreal.'),
  ('a0000000-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000001', 3, 'Great console but needs more games.'),
  ('a0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000001', 4, 'Solid hardware, fast loading.'),
  ('a0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000001', 4, 'Sturdy and looks great.'),
  ('a0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000001', 5, 'Perfect desk for home office.');
