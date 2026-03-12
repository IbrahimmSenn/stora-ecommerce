CREATE table categories (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
name VARCHAR(255) UNIQUE NOT NULL,
slug VARCHAR(255) UNIQUE NOT NULL,
parent_id UUID REFERENCES categories(id),
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


--Trigger
CREATE TRIGGER update_categories_modtime
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();