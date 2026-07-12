-- Demo category tiles reuse a representative product image from the seed
-- catalog. Guarded with image_url IS NULL so admin edits are not overwritten
-- on re-run.
UPDATE categories SET image_url = '/products/cc1b1d5b-eee4-5486-ac61-2d6d643be38c-1.webp' WHERE id = 'b7249918-2c71-5123-bd89-994bbd010919' AND image_url IS NULL; -- Electronics: MacBook Pro
UPDATE categories SET image_url = '/products/120af801-fd5f-5f5b-93a0-2c9f8d56240b-1.webp' WHERE id = 'f32ffbc4-13b1-5a7d-96a9-87de7d103657' AND image_url IS NULL; -- Beauty: Chanel Coco Noir
UPDATE categories SET image_url = '/products/0c02cf50-28c6-50e3-933f-c4f5fb771fdf-1.webp' WHERE id = 'e8b6702f-590a-58e5-bdbf-7e2a546ffde7' AND image_url IS NULL; -- Home: Table Lamp
UPDATE categories SET image_url = '/products/0c51715e-4bb0-562f-af36-4ba4a617b950-1.webp' WHERE id = '0cc9a0e7-fdd4-5340-8fe8-63635788fe34' AND image_url IS NULL; -- Furniture: Annibale Colombo Sofa
UPDATE categories SET image_url = '/products/ab348623-572b-5ff7-a9da-b1803e9eefde-1.webp' WHERE id = '5aad6047-8177-5fdc-be5b-0ca631c16552' AND image_url IS NULL; -- Clothing: Man Plaid Shirt
UPDATE categories SET image_url = '/products/94529967-44ed-5b3a-acc7-3bb865575518-1.webp' WHERE id = '49698ca8-265a-5684-a10c-502ddc8f397b' AND image_url IS NULL; -- Shoes: Nike Air Jordan 1
UPDATE categories SET image_url = '/products/f7d5f0e4-2001-5f57-8c6b-cb21d3c8652a-1.webp' WHERE id = '44fc7a48-8d3c-5239-9135-d0da818bc275' AND image_url IS NULL; -- Accessories: Rolex Datejust
UPDATE categories SET image_url = '/products/805c9e7e-dd8e-504c-a91b-1b7a576fd133-1.webp' WHERE id = 'afd4d236-4f53-5d5d-b1fb-58d5c0715d33' AND image_url IS NULL; -- Sports: Basketball
UPDATE categories SET image_url = '/products/4a757a29-c0d0-5328-8501-1ebb43528849-1.webp' WHERE id = '4dca438c-7e04-5ca2-af5a-3290074bdf5d' AND image_url IS NULL; -- Groceries: Ice Cream
UPDATE categories SET image_url = '/products/5efcc3f3-69ed-5407-a742-6d2862f5cad6-1.webp' WHERE id = '96956881-890d-54fc-83f2-16362719be91' AND image_url IS NULL; -- Automotive: Kawasaki Z800
