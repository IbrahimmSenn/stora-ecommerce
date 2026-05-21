-- Seed data — curated catalogue imported from escuelajs.co (one-shot).
-- Auth users + brands preserved; categories + products + images replaced.
-- admin@shop.com / admin123  ·  customer@shop.com / customer123  ·  test3@test.com / test123

-- Admin user
INSERT INTO users (id, email, password_hash, role) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'admin@shop.com',
   '$2b$10$eoPLJ/PW4skcSOuKY0lcuuegPRmubIawIrDGyByoK8yzn.sSGY9Qq', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Customer user
INSERT INTO users (id, email, password_hash, role) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'customer@shop.com',
   '$2b$10$/ieJ0h2MEnPkr/TH9zhvpOVZ01GS2biGBUNZ1JMspSmBiwUOhxgxG', 'customer')
ON CONFLICT (email) DO NOTHING;

-- Extra test account — handy for the Mailhog forgot-password demo.
INSERT INTO users (id, email, password_hash, role) VALUES
  ('c0000000-0000-0000-0000-000000000003', 'test3@test.com',
   '$2b$10$Y9YIZzIkLNOoYRZ.oECeKOd0efN4Tpkzp.vRv7F3xNrTBB7ilgUZG', 'customer')
ON CONFLICT (email) DO NOTHING;

-- Brands (schema demo; imported products keep brand_id NULL since the source has no brand data)
INSERT INTO brands (id, name) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'Apple'),
  ('b0000000-0000-0000-0000-000000000002', 'Samsung'),
  ('b0000000-0000-0000-0000-000000000003', 'Nike'),
  ('b0000000-0000-0000-0000-000000000004', 'Sony'),
  ('b0000000-0000-0000-0000-000000000005', 'IKEA')
ON CONFLICT (name) DO NOTHING;

-- Categories (flat — escuelajs has no hierarchy)
INSERT INTO categories (id, name, slug, parent_id) VALUES
  ('ca000000-0000-0000-0000-000000000002', 'Electronics', 'electronics', NULL),
  ('ca000000-0000-0000-0000-000000000003', 'Furniture', 'furniture', NULL),
  ('ca000000-0000-0000-0000-000000000004', 'Shoes', 'shoes', NULL),
  ('ca000000-0000-0000-0000-000000000005', 'Miscellaneous', 'miscellaneous', NULL)
ON CONFLICT (name) DO NOTHING;

-- Products (34 curated items from escuelajs.co; brand_id NULL, weight_g NULL)
INSERT INTO products (id, name, description, price, stock_quantity, category_id, brand_id, weight_g, dimensions_cm) VALUES
  ('a0000000-0000-0000-0000-000000000012', 'Sleek White & Orange Wireless Gaming Controller', 'Elevate your gaming experience with this state-of-the-art wireless controller, featuring a crisp white base with vibrant orange accents. Designed for precision play, the ergonomic shape and responsive buttons provide maximum comfort and control for endless hours of gameplay. Compatible with multiple gaming platforms, this controller is a must-have for any serious gamer looking to enhance their setup.', 6900, 183, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000013', 'Sleek Wireless Headphone & Inked Earbud Set', 'Experience the fusion of style and sound with this sophisticated audio set featuring a pair of sleek, white wireless headphones offering crystal-clear sound quality and over-ear comfort. The set also includes a set of durable earbuds, perfect for an on-the-go lifestyle. Elevate your music enjoyment with this versatile duo, designed to cater to all your listening needs.', 4400, 48, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000014', 'Sleek Comfort-Fit Over-Ear Headphones', 'Experience superior sound quality with our Sleek Comfort-Fit Over-Ear Headphones, designed for prolonged use with cushioned ear cups and an adjustable, padded headband. Ideal for immersive listening, whether you''re at home, in the office, or on the move. Their durable construction and timeless design provide both aesthetically pleasing looks and long-lasting performance.', 2800, 26, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000015', 'Efficient 2-Slice Toaster', 'Enhance your morning routine with our sleek 2-slice toaster, featuring adjustable browning controls and a removable crumb tray for easy cleaning. This compact and stylish appliance is perfect for any kitchen, ensuring your toast is always golden brown and delicious.', 4800, 90, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000016', 'Sleek Wireless Computer Mouse', 'Experience smooth and precise navigation with this modern wireless mouse, featuring a glossy finish and a comfortable ergonomic design. Its responsive tracking and easy-to-use interface make it the perfect accessory for any desktop or laptop setup. The stylish blue hue adds a splash of color to your workspace, while its compact size ensures it fits neatly in your bag for on-the-go productivity.', 1000, 82, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000017', 'Sleek Modern Laptop with Ambient Lighting', 'Experience next-level computing with our ultra-slim laptop, featuring a stunning display illuminated by ambient lighting. This high-performance machine is perfect for both work and play, delivering powerful processing in a sleek, portable design. The vibrant colors add a touch of personality to your tech collection, making it as stylish as it is functional.', 4300, 77, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000018', 'Sleek Modern Laptop for Professionals', 'Experience cutting-edge technology and elegant design with our latest laptop model. Perfect for professionals on-the-go, this high-performance laptop boasts a powerful processor, ample storage, and a long-lasting battery life, all encased in a lightweight, slim frame for ultimate portability. Shop now to elevate your work and play.', 9700, 55, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000019', 'Stylish Red & Silver Over-Ear Headphones', 'Immerse yourself in superior sound quality with these sleek red and silver over-ear headphones. Designed for comfort and style, the headphones feature cushioned ear cups, an adjustable padded headband, and a detachable red cable for easy storage and portability. Perfect for music lovers and audiophiles who value both appearance and audio fidelity.', 3900, 46, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001a', 'Sleek Mirror Finish Phone Case', 'Enhance your smartphone''s look with this ultra-sleek mirror finish phone case. Designed to offer style with protection, the case features a reflective surface that adds a touch of elegance while keeping your device safe from scratches and impacts. Perfect for those who love a minimalist and modern aesthetic.', 2700, 193, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001b', 'Sleek Smartwatch with Vibrant Display', 'Experience modern timekeeping with our high-tech smartwatch, featuring a vivid touch screen display, customizable watch faces, and a comfortable blue silicone strap. This smartwatch keeps you connected with notifications and fitness tracking while showcasing exceptional style and versatility.', 1600, 159, 'ca000000-0000-0000-0000-000000000002', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001c', 'Sleek Modern Leather Sofa', 'Enhance the elegance of your living space with our Sleek Modern Leather Sofa. Designed with a minimalist aesthetic, it features clean lines and a luxurious leather finish. The robust metal legs provide stability and support, while the plush cushions ensure comfort. Perfect for contemporary homes or office waiting areas, this sofa is a statement piece that combines style with practicality.', 5300, 42, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001d', 'Mid-Century Modern Wooden Dining Table', 'Elevate your dining room with this sleek Mid-Century Modern dining table, featuring an elegant walnut finish and tapered legs for a timeless aesthetic. Its sturdy wood construction and minimalist design make it a versatile piece that fits with a variety of decor styles. Perfect for intimate dinners or as a stylish spot for your morning coffee.', 2400, 171, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001e', 'Elegant Golden-Base Stone Top Dining Table', 'Elevate your dining space with this luxurious table, featuring a sturdy golden metal base with an intricate rod design that provides both stability and chic elegance. The smooth stone top in a sleek round shape offers a robust surface for your dining pleasure. Perfect for both everyday meals and special occasions, this table easily complements any modern or glam decor.', 6600, 128, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000001f', 'Modern Elegance Teal Armchair', 'Elevate your living space with this beautifully crafted armchair, featuring a sleek wooden frame that complements its vibrant teal upholstery. Ideal for adding a pop of color and contemporary style to any room, this chair provides both superb comfort and sophisticated design. Perfect for reading, relaxing, or creating a cozy conversation nook.', 2500, 28, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000020', 'Elegant Solid Wood Dining Table', 'Enhance your dining space with this sleek, contemporary dining table, crafted from high-quality solid wood with a warm finish. Its sturdy construction and minimalist design make it a perfect addition for any home looking for a touch of elegance. Accommodates up to six guests comfortably and includes a striking fruit bowl centerpiece. The overhead lighting is not included.', 6700, 27, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000021', 'Modern Minimalist Workstation Setup', 'Elevate your home office with our Modern Minimalist Workstation Setup, featuring a sleek wooden desk topped with an elegant computer, stylish adjustable wooden desk lamp, and complimentary accessories for a clean, productive workspace. This setup is perfect for professionals seeking a contemporary look that combines functionality with design.', 4900, 43, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000022', 'Modern Ergonomic Office Chair', 'Elevate your office space with this sleek and comfortable Modern Ergonomic Office Chair. Designed to provide optimal support throughout the workday, it features an adjustable height mechanism, smooth-rolling casters for easy mobility, and a cushioned seat for extended comfort. The clean lines and minimalist white design make it a versatile addition to any contemporary workspace.', 7100, 75, 'ca000000-0000-0000-0000-000000000003', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000023', 'Futuristic Holographic Soccer Cleats', 'Step onto the field and stand out from the crowd with these eye-catching holographic soccer cleats. Designed for the modern player, these cleats feature a sleek silhouette, lightweight construction for maximum agility, and durable studs for optimal traction. The shimmering holographic finish reflects a rainbow of colors as you move, ensuring that you''ll be noticed for both your skills and style. Perfect for the fashion-forward athlete who wants to make a statement.', 3900, 79, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000024', 'Rainbow Glitter High Heels', 'Step into the spotlight with these eye-catching rainbow glitter high heels. Designed to dazzle, each shoe boasts a kaleidoscope of shimmering colors that catch and reflect light with every step. Perfect for special occasions or a night out, these stunners are sure to turn heads and elevate any ensemble.', 3900, 149, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000025', 'Chic Summer Denim Espadrille Sandals', 'Step into summer with style in our denim espadrille sandals. Featuring a braided jute sole for a classic touch and adjustable denim straps for a snug fit, these sandals offer both comfort and a fashionable edge. The easy slip-on design ensures convenience for beach days or casual outings.', 3300, 174, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000026', 'Vibrant Runners: Bold Orange & Blue Sneakers', 'Step into style with these eye-catching sneakers featuring a striking combination of orange and blue hues. Designed for both comfort and fashion, these shoes come with flexible soles and cushioned insoles, perfect for active individuals who don''t compromise on style. The reflective silver accents add a touch of modernity, making them a standout accessory for your workout or casual wear.', 2700, 26, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000027', 'Vibrant Pink Classic Sneakers', 'Step into style with our Vibrant Pink Classic Sneakers! These eye-catching shoes feature a bold pink hue with iconic white detailing, offering a sleek, timeless design. Constructed with durable materials and a comfortable fit, they are perfect for those seeking a pop of color in their everyday footwear. Grab a pair today and add some vibrancy to your step!', 8400, 163, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000028', 'Futuristic Silver and Gold High-Top Sneaker', 'Step into the future with this eye-catching high-top sneaker, designed for those who dare to stand out. The sneaker features a sleek silver body with striking gold accents, offering a modern twist on classic footwear. Its high-top design provides support and style, making it the perfect addition to any avant-garde fashion collection. Grab a pair today and elevate your shoe game!', 6800, 70, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000029', 'Futuristic Chic High-Heel Boots', 'Elevate your style with our cutting-edge high-heel boots that blend bold design with avant-garde aesthetics. These boots feature a unique color-block heel, a sleek silhouette, and a versatile light grey finish that pairs easily with any cutting-edge outfit. Crafted for the fashion-forward individual, these boots are sure to make a statement.', 3600, 186, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002a', 'Elegant Patent Leather Peep-Toe Pumps with Gold-Tone Heel', 'Step into sophistication with these chic peep-toe pumps, showcasing a lustrous patent leather finish and an eye-catching gold-tone block heel. The ornate buckle detail adds a touch of glamour, perfect for elevating your evening attire or complementing a polished daytime look.', 5300, 199, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002b', 'Elegant Purple Leather Loafers', 'Step into sophistication with our Elegant Purple Leather Loafers, perfect for making a bold statement. Crafted from high-quality leather with a vibrant purple finish, these shoes feature a classic loafer silhouette that''s been updated with a contemporary twist. The comfortable slip-on design and durable soles ensure both style and functionality for the modern man.', 1700, 159, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002c', 'Classic Blue Suede Casual Shoes', 'Step into comfort with our Classic Blue Suede Casual Shoes, perfect for everyday wear. These shoes feature a stylish blue suede upper, durable rubber soles for superior traction, and classic lace-up fronts for a snug fit. The sleek design pairs well with both jeans and chinos, making them a versatile addition to any wardrobe.', 3900, 127, 'ca000000-0000-0000-0000-000000000004', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002d', 'Sleek Futuristic Electric Bicycle', 'This modern electric bicycle combines style and efficiency with its unique design and top-notch performance features. Equipped with a durable frame, enhanced battery life, and integrated tech capabilities, it''s perfect for the eco-conscious commuter looking to navigate the city with ease.', 2200, 76, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002e', 'Sleek All-Terrain Go-Kart', 'Experience the thrill of outdoor adventures with our Sleek All-Terrain Go-Kart, featuring a durable frame, comfortable racing seat, and robust, large-tread tires perfect for handling a variety of terrains. Designed for fun-seekers of all ages, this go-kart is an ideal choice for backyard racing or exploring local trails.', 3700, 134, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-00000000002f', 'Radiant Citrus Eau de Parfum', 'Indulge in the essence of summer with this vibrant citrus-scented Eau de Parfum. Encased in a sleek glass bottle with a bold orange cap, this fragrance embodies freshness and elegance. Perfect for daily wear, it''s an olfactory delight that leaves a lasting, zesty impression.', 7300, 170, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000030', 'Sleek Olive Green Hardshell Carry-On Luggage', 'Travel in style with our durable hardshell carry-on, perfect for weekend getaways and business trips. This sleek olive green suitcase features smooth gliding wheels for easy airport navigation, a sturdy telescopic handle, and a secure zippered compartment to keep your belongings safe. Its compact size meets most airline overhead bin requirements, ensuring a hassle-free flying experience.', 4800, 91, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000031', 'Chic Transparent Fashion Handbag', 'Elevate your style with our Chic Transparent Fashion Handbag, perfect for showcasing your essentials with a clear, modern edge. This trendy accessory features durable acrylic construction, luxe gold-tone hardware, and an elegant chain strap. Its compact size ensures you can carry your day-to-day items with ease and sophistication.', 6100, 21, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000032', 'Trendy Pink-Tinted Sunglasses', 'Step up your style game with these fashionable black-framed, pink-tinted sunglasses. Perfect for making a statement while protecting your eyes from the glare. Their bold color and contemporary design make these shades a must-have accessory for any trendsetter looking to add a pop of color to their ensemble.', 3800, 60, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL),
  ('a0000000-0000-0000-0000-000000000033', 'Elegant Glass Tumbler Set', 'Enhance your drinkware collection with our sophisticated set of glass tumblers, perfect for serving your favorite beverages. This versatile set includes both clear and subtly tinted glasses, lending a modern touch to any table setting. Crafted with quality materials, these durable tumblers are designed to withstand daily use while maintaining their elegant appeal.', 5000, 198, 'ca000000-0000-0000-0000-000000000005', NULL, NULL, NULL)
ON CONFLICT (id) DO NOTHING;

-- Product images (first image is_primary = true)
INSERT INTO product_images (product_id, url, is_primary) VALUES
  ('a0000000-0000-0000-0000-000000000012', '/products/ZANVnHE.jpeg', true),
  ('a0000000-0000-0000-0000-000000000012', '/products/Ro5z6Tn.jpeg', false),
  ('a0000000-0000-0000-0000-000000000012', '/products/woA93Li.jpeg', false),
  ('a0000000-0000-0000-0000-000000000013', '/products/yVeIeDa.jpeg', true),
  ('a0000000-0000-0000-0000-000000000013', '/products/jByJ4ih.jpeg', false),
  ('a0000000-0000-0000-0000-000000000013', '/products/KXj6Tpb.jpeg', false),
  ('a0000000-0000-0000-0000-000000000014', '/products/SolkFEB.jpeg', true),
  ('a0000000-0000-0000-0000-000000000014', '/products/KIGW49u.jpeg', false),
  ('a0000000-0000-0000-0000-000000000014', '/products/mWwek7p.jpeg', false),
  ('a0000000-0000-0000-0000-000000000015', '/products/keVCVIa.jpeg', true),
  ('a0000000-0000-0000-0000-000000000015', '/products/afHY7v2.jpeg', false),
  ('a0000000-0000-0000-0000-000000000015', '/products/yAOihUe.jpeg', false),
  ('a0000000-0000-0000-0000-000000000016', '/products/w3Y8NwQ.jpeg', true),
  ('a0000000-0000-0000-0000-000000000016', '/products/WJFOGIC.jpeg', false),
  ('a0000000-0000-0000-0000-000000000016', '/products/dV4Nklf.jpeg', false),
  ('a0000000-0000-0000-0000-000000000017', '/products/OKn1KFI.jpeg', true),
  ('a0000000-0000-0000-0000-000000000017', '/products/G4f21Ai.jpeg', false),
  ('a0000000-0000-0000-0000-000000000017', '/products/Z9oKRVJ.jpeg', false),
  ('a0000000-0000-0000-0000-000000000018', '/products/ItHcq7o.jpeg', true),
  ('a0000000-0000-0000-0000-000000000018', '/products/55GM3XZ.jpeg', false),
  ('a0000000-0000-0000-0000-000000000018', '/products/tcNJxoW.jpeg', false),
  ('a0000000-0000-0000-0000-000000000019', '/products/YaSqa06.jpeg', true),
  ('a0000000-0000-0000-0000-000000000019', '/products/isQAliJ.jpeg', false),
  ('a0000000-0000-0000-0000-000000000019', '/products/5B8UQfh.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001a', '/products/yb9UQKL.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001a', '/products/m2owtQG.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001a', '/products/bNiORct.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001b', '/products/LGk9Jn2.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001b', '/products/1ttYWaI.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001b', '/products/sPRWnJH.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001c', '/products/Qphac99.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001c', '/products/dJjpEgG.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001c', '/products/MxJyADq.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001d', '/products/DMQHGA0.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001d', '/products/qrs9QBg.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001d', '/products/XVp8T1I.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001e', '/products/NWIJKUj.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001e', '/products/Jn1YSLk.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001e', '/products/VNZRvx5.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001f', '/products/6wkyyIN.jpeg', true),
  ('a0000000-0000-0000-0000-00000000001f', '/products/Ald3Rec.jpeg', false),
  ('a0000000-0000-0000-0000-00000000001f', '/products/dIqo03c.jpeg', false),
  ('a0000000-0000-0000-0000-000000000020', '/products/4lTaHfF.jpeg', true),
  ('a0000000-0000-0000-0000-000000000020', '/products/JktHE1C.jpeg', false),
  ('a0000000-0000-0000-0000-000000000020', '/products/cQeXQMi.jpeg', false),
  ('a0000000-0000-0000-0000-000000000021', '/products/3oXNBst.jpeg', true),
  ('a0000000-0000-0000-0000-000000000021', '/products/ErYYZnT.jpeg', false),
  ('a0000000-0000-0000-0000-000000000021', '/products/boBPwYW.jpeg', false),
  ('a0000000-0000-0000-0000-000000000022', '/products/3dU0m72.jpeg', true),
  ('a0000000-0000-0000-0000-000000000022', '/products/zPU3EVa.jpeg', false),
  ('a0000000-0000-0000-0000-000000000023', '/products/qNOjJje.jpeg', true),
  ('a0000000-0000-0000-0000-000000000023', '/products/NjfCFnu.jpeg', false),
  ('a0000000-0000-0000-0000-000000000023', '/products/eYtvXS1.jpeg', false),
  ('a0000000-0000-0000-0000-000000000024', '/products/62gGzeF.jpeg', true),
  ('a0000000-0000-0000-0000-000000000024', '/products/5MoPuFM.jpeg', false),
  ('a0000000-0000-0000-0000-000000000024', '/products/sUVj7pK.jpeg', false),
  ('a0000000-0000-0000-0000-000000000025', '/products/9qrmE1b.jpeg', true),
  ('a0000000-0000-0000-0000-000000000025', '/products/wqKxBVH.jpeg', false),
  ('a0000000-0000-0000-0000-000000000025', '/products/sWSV6DK.jpeg', false),
  ('a0000000-0000-0000-0000-000000000026', '/products/hKcMNJs.jpeg', true),
  ('a0000000-0000-0000-0000-000000000026', '/products/NYToymX.jpeg', false),
  ('a0000000-0000-0000-0000-000000000026', '/products/HiiapCt.jpeg', false),
  ('a0000000-0000-0000-0000-000000000027', '/products/mcW42Gi.jpeg', true),
  ('a0000000-0000-0000-0000-000000000027', '/products/mhn7qsF.jpeg', false),
  ('a0000000-0000-0000-0000-000000000027', '/products/F8vhnFJ.jpeg', false),
  ('a0000000-0000-0000-0000-000000000028', '/products/npLfCGq.jpeg', true),
  ('a0000000-0000-0000-0000-000000000028', '/products/vYim3gj.jpeg', false),
  ('a0000000-0000-0000-0000-000000000028', '/products/HxuHwBO.jpeg', false),
  ('a0000000-0000-0000-0000-000000000029', '/products/HqYqLnW.jpeg', true),
  ('a0000000-0000-0000-0000-000000000029', '/products/RlDGnZw.jpeg', false),
  ('a0000000-0000-0000-0000-000000000029', '/products/qa0O6fg.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002a', '/products/AzAY4Ed.jpeg', true),
  ('a0000000-0000-0000-0000-00000000002a', '/products/umfnS9P.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002a', '/products/uFyuvLg.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002b', '/products/Au8J9sX.jpeg', true),
  ('a0000000-0000-0000-0000-00000000002b', '/products/gdr8BW2.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002b', '/products/KDCZxnJ.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002c', '/products/sC0ztOB.jpeg', true),
  ('a0000000-0000-0000-0000-00000000002c', '/products/Jf9DL9R.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002c', '/products/R1IN95T.jpeg', false),
  ('a0000000-0000-0000-0000-00000000002d', '/products/BG8J0Fj.jpg', true),
  ('a0000000-0000-0000-0000-00000000002d', '/products/ujHBpCX.jpg', false),
  ('a0000000-0000-0000-0000-00000000002d', '/products/WHeVL9H.jpg', false),
  ('a0000000-0000-0000-0000-00000000002e', '/products/Ex5x3IU.jpg', true),
  ('a0000000-0000-0000-0000-00000000002e', '/products/z7wAQwe.jpg', false),
  ('a0000000-0000-0000-0000-00000000002e', '/products/kc0Dj9S.jpg', false),
  ('a0000000-0000-0000-0000-00000000002f', '/products/xPDwUb3.jpg', true),
  ('a0000000-0000-0000-0000-00000000002f', '/products/3rfp691.jpg', false),
  ('a0000000-0000-0000-0000-00000000002f', '/products/kG05a29.jpg', false),
  ('a0000000-0000-0000-0000-000000000030', '/products/jVfoZnP.jpg', true),
  ('a0000000-0000-0000-0000-000000000030', '/products/Tnl15XK.jpg', false),
  ('a0000000-0000-0000-0000-000000000030', '/products/7OqTPO6.jpg', false),
  ('a0000000-0000-0000-0000-000000000031', '/products/Lqaqz59.jpg', true),
  ('a0000000-0000-0000-0000-000000000031', '/products/uSqWK0m.jpg', false),
  ('a0000000-0000-0000-0000-000000000031', '/products/atWACf1.jpg', false),
  ('a0000000-0000-0000-0000-000000000032', '/products/0qQBkxX.jpg', true),
  ('a0000000-0000-0000-0000-000000000032', '/products/I5g1DoE.jpg', false),
  ('a0000000-0000-0000-0000-000000000032', '/products/myfFQBW.jpg', false),
  ('a0000000-0000-0000-0000-000000000033', '/products/TF0pXdL.jpg', true),
  ('a0000000-0000-0000-0000-000000000033', '/products/BLDByXP.jpg', false),
  ('a0000000-0000-0000-0000-000000000033', '/products/b7trwCv.jpg', false)
ON CONFLICT (product_id, url) DO NOTHING;

