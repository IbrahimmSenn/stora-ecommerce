CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  token VARCHAR(512) NOT NULL UNIQUE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  --SECURITY FLAGS
  revoked BOOLEAN DEFAULT false NOT NULL,
  used BOOLEAN DEFAULT false NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);

CREATE TRIGGER update_refresh_tokens_modtime
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();