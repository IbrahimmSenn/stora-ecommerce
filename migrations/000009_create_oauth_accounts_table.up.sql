CREATE TABLE oauth_accounts(
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider VARCHAR(50) NOT NULL,
  provider_user_id VARCHAR(255) NOT NULL,
  UNIQUE (provider, provider_user_id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


CREATE TRIGGER update_oauth_accounts_modtime
BEFORE UPDATE ON oauth_accounts
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();