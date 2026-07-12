package config

import (
	"strings"
	"testing"
)

func goodProdConfig() *Config {
	return &Config{
		AppEnv:               "production",
		SkipCaptcha:          false,
		JWTSecret:            "this-is-a-sufficiently-long-secret-key",
		EncryptionKey:        "8c6f2a1d4e9b7c3f0a5d8e2b6c9f1a4d7e0b3c6f9a2d5e8b1c4f7a0d3e6b9c2f",
		BaseURL:              "https://shop.example.com",
		RabbitMQURL:          "amqp://app:s3cret@rabbitmq:5672/",
		DatabaseURL:          "postgres://app:s3cret@db:5432/shop?sslmode=require",
		StripeSecretKey:      "sk_live_abcdefghijklmnop",
		StripePublishableKey: "pk_live_abcdefghijklmnop",
		StripeWebhookSecret:  "whsec_abcdefghijklmnop123456",
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
		"stripe test-mode secret key": func(c *Config) string {
			c.StripeSecretKey = "sk_test_abcdefghijklmnop"
			return "https://shop.example.com"
		},
		"stripe placeholder secret key": func(c *Config) string {
			c.StripeSecretKey = "sk_test_x"
			return "https://shop.example.com"
		},
		"stripe test-mode publishable key": func(c *Config) string {
			c.StripePublishableKey = "pk_test_abcdefghijklmnop"
			return "https://shop.example.com"
		},
		"stripe placeholder webhook secret": func(c *Config) string {
			c.StripeWebhookSecret = "whsec_x"
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

func goodDemoConfig() *Config {
	c := goodProdConfig()
	c.DemoMode = true
	c.AdminPassword = "a-long-demo-admin-password"
	c.StripeSecretKey = "sk_test_abcdefghijklmnop"
	c.StripePublishableKey = "pk_test_abcdefghijklmnop"
	c.DatabaseURL = "postgres://app:s3cret@db:5432/shop?sslmode=disable"
	return c
}

// Demo mode = public portfolio deployment: Stripe test keys and network-internal
// Postgres are fine, everything else stays production-strict.
func TestValidateForDemo_AllowsTestStripeKeysAndInternalDB(t *testing.T) {
	if err := goodDemoConfig().validateForProduction("https://shop.example.com"); err != nil {
		t.Fatalf("expected sound demo config to pass, got: %v", err)
	}
}

func TestValidateForDemo_RejectsInsecure(t *testing.T) {
	cases := map[string]func(*Config) (cors string){
		"missing admin password": func(c *Config) string { c.AdminPassword = ""; return "https://shop.example.com" },
		"short admin password":   func(c *Config) string { c.AdminPassword = "admin123"; return "https://shop.example.com" },
		"stripe placeholder secret key": func(c *Config) string {
			c.StripeSecretKey = "sk_test_x"
			return "https://shop.example.com"
		},
		"stripe placeholder publishable key": func(c *Config) string {
			c.StripePublishableKey = "pk_test_x"
			return "https://shop.example.com"
		},
		"placeholder webhook secret": func(c *Config) string {
			c.StripeWebhookSecret = "whsec_x"
			return "https://shop.example.com"
		},
		"skip captcha": func(c *Config) string { c.SkipCaptcha = true; return "https://shop.example.com" },
		"default rabbit": func(c *Config) string {
			c.RabbitMQURL = "amqp://guest:guest@rabbitmq:5672/"
			return "https://shop.example.com"
		},
		"db default creds": func(c *Config) string {
			c.DatabaseURL = "postgres://admin:secret@db/shop?sslmode=disable"
			return "https://shop.example.com"
		},
		"placeholder jwt": func(c *Config) string { c.JWTSecret = placeholderJWTSecret; return "https://shop.example.com" },
		"non-https base":  func(c *Config) string { c.BaseURL = "http://shop.example.com"; return "https://shop.example.com" },
		"missing cors":    func(c *Config) string { return "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := goodDemoConfig()
			cors := mutate(c)
			if err := c.validateForProduction(cors); err == nil {
				t.Fatalf("expected %q to be rejected in demo mode", name)
			}
		})
	}
}

// All failures must surface in one pass, not one restart at a time.
func TestValidateForProduction_ReportsAllFailures(t *testing.T) {
	c := goodProdConfig()
	c.JWTSecret = "tooshort"
	c.StripeSecretKey = "sk_test_x"
	c.BaseURL = "http://shop.example.com"

	err := c.validateForProduction("https://shop.example.com")
	if err == nil {
		t.Fatal("expected errors for broken prod config")
	}
	for _, want := range []string{"JWT_SECRET", "STRIPE_SECRET_KEY", "BASE_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s, got:\n%v", want, err)
		}
	}
}
