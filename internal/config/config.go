// config.go — loads settings from environment variables / .env file.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

	// RabbitMQ
	RabbitMQURL string

	// Nominatim (OpenStreetMap) address verification.
	// NominatimUserAgent is required by OSM usage policy — empty disables
	// the check (falls back to a passthrough geocoder).
	NominatimBaseURL   string
	NominatimUserAgent string

	// CookieSecure marks auth/session cookies with the Secure attribute so
	// browsers only send them over HTTPS. Enabled when APP_ENV=production.
	CookieSecure bool

	// AppEnv is the raw APP_ENV value ("production" enables stricter checks).
	AppEnv string

	// DemoMode marks a public portfolio deployment: production-strict security,
	// but Stripe stays in test mode and the demo catalogue/users are seeded.
	// Requires ADMIN_PASSWORD so the seeded admin never keeps the known dev
	// password on a public host.
	DemoMode      bool
	AdminPassword string

	// UploadDir is where uploaded product images + generated variants are
	// written; served under /media. Defaults to ./uploads.
	UploadDir string

	// TLS — when enabled, an HTTPS listener runs alongside HTTP. A self-signed
	// cert is generated at the cert/key paths if they don't exist.
	TLSEnabled  bool
	TLSPort     string
	TLSCertFile string
	TLSKeyFile  string

	// CORSOrigins is the allow-list of browser origins. The SPA is served
	// same-origin (or proxied in dev), so this only governs cross-origin
	// callers — never wildcard with credentials.
	CORSOrigins []string

	// Rate limiting (per client IP). General is the global safety net; Auth
	// is the strict bucket guarding brute-forceable auth endpoints. All
	// tunable via env so load tests can relax them.
	RateLimitRPS       float64
	RateLimitBurst     int
	AuthRateLimitRPS   float64
	AuthRateLimitBurst int

	// Database connection pool sizing. pgxpool defaults to 4×GOMAXPROCS max
	// conns with no floor, which starves first under load — size it explicitly.
	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	// RedisURL is optional. Empty = single-binary mode (in-memory rate limiting
	// and cache). Set it to share rate-limit state and the read cache across
	// multiple app instances behind a load balancer.
	RedisURL string

	// MetricsAddr is the internal listener serving /metrics for Prometheus.
	// Not published outside the compose network.
	MetricsAddr string

	// LogFormat overrides the APP_ENV-derived log handler: "json" or "text".
	// Empty keeps the default (JSON in production, text otherwise).
	LogFormat string

	// OTelEnabled turns on distributed tracing (spans exported over OTLP/HTTP
	// to OTelEndpoint — Tempo in the monitoring profile). Off by default; the
	// app runs identically with a no-op tracer.
	OTelEnabled  bool
	OTelEndpoint string
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

		RabbitMQURL: os.Getenv("RABBITMQ_URL"),

		NominatimBaseURL:   os.Getenv("NOMINATIM_BASE_URL"),
		NominatimUserAgent: os.Getenv("NOMINATIM_USER_AGENT"),

		AppEnv:        os.Getenv("APP_ENV"),
		DemoMode:      os.Getenv("DEMO_MODE") == "true",
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		UploadDir:     os.Getenv("UPLOAD_DIR"),
		CookieSecure:  os.Getenv("APP_ENV") == "production",

		TLSEnabled:  os.Getenv("TLS_ENABLED") == "true",
		TLSPort:     os.Getenv("TLS_PORT"),
		TLSCertFile: os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TLS_KEY_FILE"),

		RateLimitRPS:       envFloat("RATE_LIMIT_RPS", 30),
		RateLimitBurst:     envInt("RATE_LIMIT_BURST", 60),
		AuthRateLimitRPS:   envFloat("AUTH_RATE_LIMIT_RPS", 0.2),
		AuthRateLimitBurst: envInt("AUTH_RATE_LIMIT_BURST", 8),

		DBMaxConns:        envInt("DB_MAX_CONNS", 25),
		DBMinConns:        envInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime: time.Duration(envInt("DB_MAX_CONN_LIFETIME_MIN", 60)) * time.Minute,
		DBMaxConnIdleTime: time.Duration(envInt("DB_MAX_CONN_IDLE_MIN", 5)) * time.Minute,

		RedisURL: os.Getenv("REDIS_URL"),

		MetricsAddr: os.Getenv("METRICS_ADDR"),
		LogFormat:   os.Getenv("LOG_FORMAT"),

		OTelEnabled:  os.Getenv("OTEL_ENABLED") == "true",
		OTelEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	// CORS origins: explicit allow-list. Defaults to the local dev origins so
	// nothing breaks out of the box (the SPA itself is same-origin and never
	// needs CORS); production must set CORS_ORIGINS to the real domain.
	corsRaw := os.Getenv("CORS_ORIGINS")
	if corsRaw == "" {
		cfg.CORSOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	} else {
		for _, o := range strings.Split(corsRaw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, o)
			}
		}
	}

	if cfg.NominatimBaseURL == "" {
		cfg.NominatimBaseURL = "https://nominatim.openstreetmap.org"
	}

	if cfg.NominatimUserAgent == "" {
		cfg.NominatimUserAgent = "stora/1.0 (+https://github.com/IbrahimmSenn/stora-ecommerce)"
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

	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required (e.g. amqp://guest:guest@rabbitmq:5672/)")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}

	if cfg.TLSPort == "" {
		cfg.TLSPort = "8443"
	}
	if cfg.TLSCertFile == "" {
		cfg.TLSCertFile = "./certs/server.crt"
	}
	if cfg.TLSKeyFile == "" {
		cfg.TLSKeyFile = "./certs/server.key"
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:" + cfg.Port
	}

	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}

	if cfg.MetricsAddr == "" {
		cfg.MetricsAddr = ":9091"
	}

	if cfg.OTelEndpoint == "" {
		cfg.OTelEndpoint = "http://tempo:4318" // compose service; only used when OTEL_ENABLED=true
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

	if err := cfg.validateForProduction(corsRaw); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Publicly-known placeholder values shipped in .env.example. Booting production
// with either would silently run with secrets anyone can read from the repo.
const (
	placeholderJWTSecret = "replace-with-a-random-string-at-least-32-chars"
	devEncryptionKey     = "0000000000000000000000000000000000000000000000000000000000000000"
)

// validateForProduction fails fast on insecure settings when APP_ENV=production.
// In any other environment these are warnings at most, so local dev is never
// blocked. All failures are collected and returned together so the operator can
// fix the whole .env in one pass. corsExplicit is the raw CORS_ORIGINS value
// (empty = using defaults).
func (c *Config) validateForProduction(corsExplicit string) error {
	if c.AppEnv != "production" {
		return nil
	}
	var errs []error
	if c.SkipCaptcha {
		errs = append(errs, fmt.Errorf("SKIP_CAPTCHA must not be true in production"))
	}
	if len(c.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least 32 chars in production (got %d)", len(c.JWTSecret)))
	}
	if c.JWTSecret == placeholderJWTSecret {
		errs = append(errs, fmt.Errorf("JWT_SECRET is still the .env.example placeholder — generate one with: openssl rand -hex 32"))
	}
	if c.EncryptionKey == devEncryptionKey {
		errs = append(errs, fmt.Errorf("ENCRYPTION_KEY is still the all-zero dev key from .env.example — generate one with: openssl rand -hex 32"))
	}
	if !strings.HasPrefix(c.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("BASE_URL must be https:// in production"))
	}
	if corsExplicit == "" {
		errs = append(errs, fmt.Errorf("CORS_ORIGINS must be set to your frontend origin(s) in production"))
	}
	if strings.Contains(c.RabbitMQURL, "guest:guest@") {
		errs = append(errs, fmt.Errorf("RABBITMQ_URL must not use the default guest:guest credentials in production"))
	}
	if strings.Contains(c.DatabaseURL, ":secret@") {
		errs = append(errs, fmt.Errorf("DATABASE_URL must not use the default dev password in production"))
	}
	// Postgres runs on the private compose network in demo deployments, where
	// the alpine image has no TLS — only full production requires sslmode.
	if !c.DemoMode && strings.Contains(c.DatabaseURL, "sslmode=disable") {
		errs = append(errs, fmt.Errorf("DATABASE_URL must use sslmode=require in production"))
	}
	// Live-mode Stripe keys are sk_live_/rk_live_ (restricted) and pk_live_.
	// Test-mode keys or the CI placeholders (sk_test_x etc.) in production mean
	// the shop looks deployed but can never take a real payment. DEMO_MODE
	// deliberately runs Stripe in test mode, so real test keys are accepted;
	// the webhook secret must still be a dashboard-configured whsec_ either way.
	if c.DemoMode {
		if !hasStripeKey(c.StripeSecretKey, "sk_live_", "rk_live_", "sk_test_", "rk_test_") {
			errs = append(errs, fmt.Errorf("STRIPE_SECRET_KEY must be a real Stripe key (sk_test_/sk_live_) in demo mode, got a placeholder"))
		}
		if !hasStripeKey(c.StripePublishableKey, "pk_live_", "pk_test_") {
			errs = append(errs, fmt.Errorf("STRIPE_PUBLISHABLE_KEY must be a real Stripe key (pk_test_/pk_live_) in demo mode, got a placeholder"))
		}
		if c.AdminPassword == "" {
			errs = append(errs, fmt.Errorf("ADMIN_PASSWORD is required in demo mode — the seeded admin must not keep the dev password on a public host"))
		} else if len(c.AdminPassword) < 12 {
			// Also rules out the publicly-known dev password admin123.
			errs = append(errs, fmt.Errorf("ADMIN_PASSWORD must be at least 12 chars in demo mode (got %d)", len(c.AdminPassword)))
		}
	} else {
		if !strings.HasPrefix(c.StripeSecretKey, "sk_live_") && !strings.HasPrefix(c.StripeSecretKey, "rk_live_") {
			errs = append(errs, fmt.Errorf("STRIPE_SECRET_KEY must be a live-mode key (sk_live_/rk_live_) in production, got a test or placeholder key"))
		}
		if !strings.HasPrefix(c.StripePublishableKey, "pk_live_") {
			errs = append(errs, fmt.Errorf("STRIPE_PUBLISHABLE_KEY must be a live-mode key (pk_live_) in production, got a test or placeholder key"))
		}
	}
	if !strings.HasPrefix(c.StripeWebhookSecret, "whsec_") || len(c.StripeWebhookSecret) < 20 {
		errs = append(errs, fmt.Errorf("STRIPE_WEBHOOK_SECRET must be a real whsec_ signing secret in production (from the dashboard webhook endpoint, not `stripe listen`)"))
	}
	return errors.Join(errs...)
}

// hasStripeKey reports whether key starts with one of the given prefixes and
// carries a real key body after it (filters CI placeholders like "sk_test_x").
func hasStripeKey(key string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) && len(key) >= len(p)+16 {
			return true
		}
	}
	return false
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
