// config.go — loads settings from environment variables / .env file.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	BcryptCost    int
	JWTSecret     string
	EncryptionKey string
	Port          string
	BaseURL       string

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	FBClientID         string
	FBClientSecret     string

	// reCAPTCHA v3
	RecaptchaSiteKey   string
	RecaptchaSecretKey string
	SkipCaptcha        bool

	// SMTP (password recovery + order emails)
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Stripe
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
		Port:          os.Getenv("PORT"),
		BaseURL:       os.Getenv("BASE_URL"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		FBClientID:         os.Getenv("FB_CLIENT_ID"),
		FBClientSecret:     os.Getenv("FB_CLIENT_SECRET"),

		RecaptchaSiteKey:   os.Getenv("RECAPTCHA_SITE_KEY"),
		RecaptchaSecretKey: os.Getenv("RECAPTCHA_SECRET_KEY"),
		SkipCaptcha:        os.Getenv("SKIP_CAPTCHA") == "true",

		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: os.Getenv("SMTP_PORT"),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
		SMTPFrom: os.Getenv("SMTP_FROM"),

		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required (generate with: openssl rand -hex 32)")
	}
	if len(cfg.EncryptionKey) != 64 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 64 hex chars (32 bytes), got %d", len(cfg.EncryptionKey))
	}

	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is required (sk_test_... from https://dashboard.stripe.com/test/apikeys)")
	}
	if cfg.StripeWebhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required (whsec_... shown by `stripe listen`)")
	}
	if cfg.StripePublishableKey == "" {
		return nil, fmt.Errorf("STRIPE_PUBLISHABLE_KEY is required (pk_test_...)")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:" + cfg.Port
	}

	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}

	if cfg.SMTPFrom == "" && cfg.SMTPUser != "" {
		cfg.SMTPFrom = cfg.SMTPUser
	}

	costStr := os.Getenv("BCRYPT_COST")
	if costStr == "" {
		cfg.BcryptCost = 10
	} else {
		cost, err := strconv.Atoi(costStr)
		if err != nil {
			return nil, fmt.Errorf("BCRYPT_COST must be a valid integer: %w", err)
		}
		cfg.BcryptCost = cost
	}

	return cfg, nil
}
