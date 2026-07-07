package config

import "testing"

func goodProdConfig() *Config {
	return &Config{
		AppEnv:        "production",
		SkipCaptcha:   false,
		JWTSecret:     "this-is-a-sufficiently-long-secret-key",
		EncryptionKey: "8c6f2a1d4e9b7c3f0a5d8e2b6c9f1a4d7e0b3c6f9a2d5e8b1c4f7a0d3e6b9c2f",
		BaseURL:       "https://shop.example.com",
		RabbitMQURL:   "amqp://app:s3cret@rabbitmq:5672/",
		DatabaseURL:   "postgres://app:s3cret@db:5432/shop?sslmode=require",
	}
}

func TestValidateForProduction_PassesWhenSound(t *testing.T) {
	if err := goodProdConfig().validateForProduction("https://shop.example.com"); err != nil {
		t.Fatalf("expected sound prod config to pass, got: %v", err)
	}
}

func TestValidateForProduction_SkippedOutsideProduction(t *testing.T) {
	c := &Config{AppEnv: "development", SkipCaptcha: true, JWTSecret: "short", BaseURL: "http://x"}
	if err := c.validateForProduction(""); err != nil {
		t.Fatalf("non-production must never be blocked, got: %v", err)
	}
}

func TestValidateForProduction_RejectsInsecure(t *testing.T) {
	cases := map[string]func(*Config) (cors string){
		"skip captcha": func(c *Config) string { c.SkipCaptcha = true; return "https://shop.example.com" },
		"short jwt":    func(c *Config) string { c.JWTSecret = "tooshort"; return "https://shop.example.com" },
		"placeholder jwt": func(c *Config) string {
			c.JWTSecret = placeholderJWTSecret
			return "https://shop.example.com"
		},
		"all-zero encryption key": func(c *Config) string {
			c.EncryptionKey = devEncryptionKey
			return "https://shop.example.com"
		},
		"non-https base": func(c *Config) string { c.BaseURL = "http://shop.example.com"; return "https://shop.example.com" },
		"missing cors":   func(c *Config) string { return "" },
		"default rabbit": func(c *Config) string {
			c.RabbitMQURL = "amqp://guest:guest@rabbitmq:5672/"
			return "https://shop.example.com"
		},
		"db sslmode off": func(c *Config) string {
			c.DatabaseURL = "postgres://app:p@db/shop?sslmode=disable"
			return "https://shop.example.com"
		},
		"db default creds": func(c *Config) string {
			c.DatabaseURL = "postgres://admin:secret@db/shop?sslmode=require"
			return "https://shop.example.com"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := goodProdConfig()
			cors := mutate(c)
			if err := c.validateForProduction(cors); err == nil {
				t.Fatalf("expected %q to be rejected in production", name)
			}
		})
	}
}
