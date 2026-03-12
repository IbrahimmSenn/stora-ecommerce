CREATE table brands (
id UUID PRIMARY KEY  DEFAULT gen_random_uuid(),
name VARCHAR(255) UNIQUE NOT NULL,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

--Trigger
CREATE TRIGGER update_brands_modtime
BEFORE UPDATE ON brands
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();