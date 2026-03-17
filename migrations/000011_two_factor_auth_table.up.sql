CREATE TABLE two_factor_auth (
  id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  secret_key TEXT NOT NULL,
  is_enabled BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);