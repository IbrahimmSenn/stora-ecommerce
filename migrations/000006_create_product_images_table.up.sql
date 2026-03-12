CREATE TABLE product_images (
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  product_id UUID REFERENCES products(id),
  url TEXT NOT NULL, --AWS3 or local i need to choose later on
  is_primary BOOLEAN DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 


CREATE TRIGGER update_product_images_modtime
BEFORE UPDATE ON product_images
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();