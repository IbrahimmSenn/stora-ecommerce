CREATE table products (
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  name VARCHAR(255) NOT NULL,
  description TEXT, --MAYBE ADD NOT NULL
  price int NOT NULL CHECK(price >= 0), -- FASTER, TAKES LESS SPACE TO USE INT INSTEAD OF NUMERIC
  stock_quantity int NOT NULL CHECK(stock_quantity >= 0),
  category_id UUID REFERENCES categories(id),
  brand_id UUID REFERENCES brands(id),
  --image_id UUID REFERENCES images (id),
  weight_g int NOT NULL CHECK(weight_g >= 0),
  weight_oz INTEGER GENERATED ALWAYS AS (ROUND(weight_g * 0.035274)) STORED,
  dimensions_cm NUMERIC(8,2) CHECK(dimensions_cm >= 0),
  dimensions_inch NUMERIC(8,2) GENERATED ALWAYS AS (ROUND(dimensions_cm * 0.393701, 2)) STORED,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


CREATE TRIGGER update_products_modtime
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();