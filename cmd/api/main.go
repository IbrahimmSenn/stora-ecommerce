// main.go — entrypoint. Connects to postgres, wires up all packages, and starts the server.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v76"
	"golang.org/x/crypto/bcrypt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/activity"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/address"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/audit"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/auth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/brand"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cache"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/captcha"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cart"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/category"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/config"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/contact"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/delivery"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/imageproc"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/mailer"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/messaging"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/metrics"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/notifications"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/oauth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/payments"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/recommend"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/review"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/seed"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/seo"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/tlsutil"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/tracing"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/vitals"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	setupLogging(cfg.AppEnv, cfg.LogFormat)

	// Distributed tracing (OTLP -> Tempo). No-op unless OTEL_ENABLED=true.
	shutdownTracing, err := tracing.Setup(context.Background(), cfg.OTelEnabled, cfg.OTelEndpoint)
	if err != nil {
		log.Fatalf("tracing setup: %v", err)
	}
	if cfg.OTelEnabled {
		log.Printf("tracing enabled (OTLP -> %s)", cfg.OTelEndpoint)
	}

	// Prometheus registry + the Recorder services use for domain counters.
	// Served on the internal metrics listener, scraped by the monitoring stack.
	promReg := metrics.NewRegistry()
	rec := metrics.NewProm(promReg)
	httpMetrics := metrics.NewHTTPMetrics(promReg)
	if cfg.OTelEnabled {
		// Latency histogram samples carry the trace ID as an exemplar so
		// Grafana can jump from a latency spike to the exact trace.
		httpMetrics.ExemplarFn = tracing.ExemplarTraceID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("parse database url: %v", err)
	}
	poolCfg.MaxConns = int32(cfg.DBMaxConns) // #nosec G115 -- operator-set env value, small
	poolCfg.MinConns = int32(cfg.DBMinConns) // #nosec G115 -- operator-set env value, small
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolCfg.HealthCheckPeriod = time.Minute
	// pgx takes one tracer; WithPgxTracing composes the Prometheus tracer
	// with OTel query spans when tracing is on.
	poolCfg.ConnConfig.Tracer = tracing.WithPgxTracing(metrics.NewQueryTracer(promReg), cfg.OTelEnabled)

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()
	promReg.MustRegister(metrics.NewPoolStatsCollector(db))

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	log.Println("connected to database")

	// --- Optional Redis (shared rate-limit + cache across instances) ---
	// Default is single-binary: in-memory cache and per-instance rate limiting.
	// Setting REDIS_URL shares both across horizontally-scaled instances behind
	// a load balancer — the seam that makes this app ready to scale out.
	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opt, perr := redis.ParseURL(cfg.RedisURL)
		if perr != nil {
			log.Fatalf("parse redis url: %v", perr)
		}
		rdb = redis.NewClient(opt)
		if perr := rdb.Ping(ctx).Err(); perr != nil {
			log.Fatalf("ping redis: %v", perr)
		}
		defer rdb.Close()
		log.Println("connected to redis (shared rate-limit + cache)")
	}

	var appCache cache.Cache
	if rdb != nil {
		appCache = cache.NewRedis(rdb, "cache:")
	} else {
		appCache = cache.NewMemory(time.Minute)
	}

	// --- Dependency wiring ---

	captchaVerifier := captcha.NewVerifier(cfg.RecaptchaSecretKey, cfg.SkipCaptcha)
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("init encryptor: %v", err)
	}

	// Demo users (encrypted email) + their reviews are seeded by the app outside
	// production, since AES-GCM email ciphertext can't be produced in seed.sql.
	// DEMO_MODE deployments also seed, with the admin password from env.
	if cfg.AppEnv != "production" || cfg.DemoMode {
		adminHash := ""
		if cfg.AdminPassword != "" {
			h, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), cfg.BcryptCost)
			if err != nil {
				log.Fatalf("hash ADMIN_PASSWORD: %v", err)
			}
			adminHash = string(h)
		}
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		seed.Demo(seedCtx, db, encryptor, adminHash)
		seedCancel()
	}

	stripe.Key = cfg.StripeSecretKey

	// --- RabbitMQ connection + topology ---
	// Dial with bounded retry to absorb the brief window between rabbitmq's
	// healthcheck going green and the broker being fully ready to serve.
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	amqpConn, err := messaging.Connect(dialCtx, cfg.RabbitMQURL)
	dialCancel()
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer amqpConn.Close()

	amqpPublisher, err := messaging.NewPublisher(amqpConn)
	if err != nil {
		log.Fatalf("open publisher channel: %v", err)
	}
	defer amqpPublisher.Close()

	if err := messaging.DeclarePaymentsTopology(amqpPublisher.Channel()); err != nil {
		log.Fatalf("declare topology: %v", err)
	}

	paymentsEventPublisher := payments.NewAmqpPublisher(amqpPublisher)

	userRepo := user.NewUserRepository(db, encryptor)
	userService := user.NewService(userRepo, cfg.BcryptCost, captchaVerifier)
	userHandler := user.NewHandler(userService)

	authRepo := auth.NewAuthRepository(db, encryptor)
	authService := auth.NewService(userRepo, authRepo, cfg.JWTSecret,
		auth.WithMailer(mail),
		auth.WithBaseURL(cfg.BaseURL),
		auth.WithBcryptCost(cfg.BcryptCost),
		auth.WithMetrics(rec),
	)
	authHandler := auth.NewHandler(authService, cfg.CookieSecure)

	auditRecorder := audit.NewRecorder(db, encryptor)

	// 2FA enforcement for staff: resolves whether a user has 2FA enabled.
	// Injected into RequireStaff2FA so middleware stays decoupled from auth.
	twoFactorChecker := mw.TwoFactorChecker(func(ctx context.Context, userID string) (bool, error) {
		tfa, err := authRepo.Get2FAByUserID(ctx, userID)
		if errors.Is(err, auth.Err2FANotEnabled) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return tfa.IsEnabled, nil
	})

	contactHandler := contact.NewHandler(contact.NewService(mail, cfg.SMTPFrom))

	addressHandler := address.NewHandler(address.NewService(address.NewRepository(db, encryptor)))

	brandRepo := brand.NewRepository(db)
	brandService := brand.NewService(brandRepo)
	brandHandler := brand.NewHandler(brandService)

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewServiceWithCache(categoryRepo, appCache, time.Minute)
	categoryHandler := category.NewHandler(categoryService)

	deliveryRepo := delivery.NewRepository(db)
	deliveryService := delivery.NewService(deliveryRepo)
	deliveryHandler := delivery.NewHandler(deliveryService)

	activityRepo := activity.NewRepository(db)
	activityService := activity.NewService(activityRepo)

	imageProcessor, err := imageproc.New(cfg.UploadDir, "/media")
	if err != nil {
		log.Fatalf("init image processor: %v", err)
	}

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo, product.WithImageProcessor(imageProcessor))
	productHandler := product.NewHandler(productService, activityService)

	reviewRepo := review.NewRepository(db)
	reviewService := review.NewService(reviewRepo)
	reviewHandler := review.NewHandler(reviewService)

	cartRepo := cart.NewRepository(db)
	cartService := cart.NewService(cartRepo, productRepo, activityService)
	cartHandler := cart.NewHandler(cartService, cfg.CookieSecure)

	recommendService := recommend.NewService(activityService, productRepo)
	recommendHandler := recommend.NewHandler(recommendService, cartService)

	// Break the orders ↔ payments import cycle: orders calls payments for
	// refunds and for reconciling stuck-pending orders against Stripe, so we
	// declare the variable here and let closure adapters resolve it once
	// payments is built below.
	var paymentsService payments.Service
	refunder := orders.RefunderFunc(func(ctx context.Context, orderID uuid.UUID) error {
		return paymentsService.RefundOrder(ctx, orderID)
	})
	reconciler := orders.ReconcilerFunc(func(ctx context.Context, orderID uuid.UUID) error {
		if paymentsService == nil {
			return nil
		}
		return paymentsService.Reconcile(ctx, orderID)
	})

	geocoder := orders.NewNominatimGeocoder(cfg.NominatimBaseURL, cfg.NominatimUserAgent)
	log.Printf("address verification: nominatim at %s", cfg.NominatimBaseURL)

	ordersRepo := orders.NewRepository(db)
	ordersService := orders.NewService(ordersRepo, cartService, encryptor, geocoder, refunder, reconciler, deliveryService,
		orders.WithMetrics(rec), orders.WithActivityLogger(activityService))
	ordersHandler := orders.NewHandler(ordersService)

	paymentsRepo := payments.NewRepository(db, encryptor)
	paymentsService = payments.NewService(
		paymentsRepo, ordersService, paymentsEventPublisher,
		payments.NewStripeClient(),
		payments.NewStripeRefundClient(),
		cfg.StripeWebhookSecret, cfg.StripePublishableKey,
		payments.WithMetrics(rec),
	)
	paymentsHandler := payments.NewHandler(paymentsService)

	// --- Notifications consumer (subscribes to payment events) ---
	emailConsumer := &notifications.EmailConsumer{Orders: ordersService, Mail: mail}
	amqpConsumer, err := messaging.NewConsumer(amqpConn, messaging.QueuePaymentEmails)
	if err != nil {
		log.Fatalf("open consumer channel: %v", err)
	}
	defer amqpConsumer.Close()

	consumerCtx, stopConsumer := context.WithCancel(context.Background())
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		if err := amqpConsumer.Run(consumerCtx, emailConsumer.Handle); err != nil && consumerCtx.Err() == nil {
			log.Printf("consumer exited unexpectedly: %v", err)
		}
	}()

	// --- OAuth providers ---
	// Token generation and storage are injected as closures to keep oauth
	// and auth packages decoupled (otherwise they'd have a circular import).

	oauthRepo := oauth.NewRepository(db)

	storeRefresh := func(ctx context.Context, token string, userID uuid.UUID) error {
		rt := auth.RefreshToken{
			ID:        uuid.New(),
			Token:     token,
			UserID:    userID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		return authRepo.StoreRefreshToken(ctx, rt)
	}

	generateJWT := func(userID, email, role, secret string) (string, string, error) {
		pair, err := auth.GenerateTokenPair(userID, email, role, secret)
		if err != nil {
			return "", "", err
		}
		return pair.AccessToken, pair.RefreshToken, nil
	}

	oauthService := oauth.NewService(userRepo, oauthRepo, generateJWT, storeRefresh, cfg.JWTSecret)

	providers := make(map[string]oauth.Provider)
	if cfg.GoogleClientID != "" {
		providers["google"] = oauth.NewGoogle(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.BaseURL+"/api/v1/auth/oauth/google/callback",
		)
	}
	if cfg.FBClientID != "" {
		providers["facebook"] = oauth.NewFacebook(
			cfg.FBClientID,
			cfg.FBClientSecret,
			cfg.BaseURL+"/api/v1/auth/oauth/facebook/callback",
		)
	}

	oauthHandler := oauth.NewHandler(oauthService, providers, cfg.BaseURL, cfg.CookieSecure)

	// --- Token validator for middleware ---
	// Same idea as above — keeps middleware decoupled from auth.

	tokenValidator := mw.TokenValidator(func(tokenString string) (*mw.TokenClaims, error) {
		claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			return nil, err
		}
		return &mw.TokenClaims{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}, nil
	})

	// --- Router ---

	r := chi.NewRouter()

	// tracing outermost so every later stage (metrics exemplars, access log
	// trace_id, handlers, pgx spans) sees the span context; httpMetrics next
	// so it times the full middleware stack; AccessLog replaces chi's
	// plain-text Logger with slog (JSON under LOG_FORMAT=json, which is what
	// Promtail ships to Loki).
	r.Use(tracing.Middleware)
	r.Use(httpMetrics.Middleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.AccessLog)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	// HSTS only in production (behind real HTTPS) — sending it over the dev
	// HTTP listener would pin browsers to https://localhost and break local dev.
	r.Use(mw.SecurityHeaders(cfg.AppEnv == "production"))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Rate limiting (token bucket, per client IP) ---
	// A loose limiter is the safety net for the API; a strict limiter guards the
	// brute-forceable auth endpoints. RealIP (above) has already resolved the
	// client address into RemoteAddr. Limits are env-tunable (e.g. relaxed for
	// load tests) via RATE_LIMIT_* / AUTH_RATE_LIMIT_*.
	//
	// Scoped to /api/* so static SPA assets and /media product images aren't
	// throttled (an image-heavy page would otherwise burn the bucket and 429),
	// and the Stripe webhook is exempt so provider retries are never dropped.
	var generalLimiter, authLimiter *mw.RateLimiter
	if rdb != nil {
		// Shared across instances: one global budget per client IP.
		generalLimiter = mw.NewRedisRateLimiter(rdb, "rl:gen:", cfg.RateLimitRPS, cfg.RateLimitBurst)
		authLimiter = mw.NewRedisRateLimiter(rdb, "rl:auth:", cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
	} else {
		generalLimiter = mw.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
		authLimiter = mw.NewRateLimiter(cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
	}
	generalLimiter.Instrument("general", func() { rec.RateLimited("general") })
	authLimiter.Instrument("auth", func() { rec.RateLimited("auth") })
	r.Use(mw.ScopePath(generalLimiter.Middleware, "/api/", "/api/v1/webhooks/stripe"))

	// --- Frontend (React SPA built into web/dist) ---
	// Static assets and the SPA fallback share one handler below — see the
	// NotFound block at the bottom of the router.
	webDist := "web/dist"

	// Expose reCAPTCHA site key to frontend (public, non-secret). When captcha
	// is skipped, advertise no key so the frontend omits the token instead of
	// loading Google's script (which the CSP blocks).
	r.Get("/api/v1/config/recaptcha", func(w http.ResponseWriter, r *http.Request) {
		siteKey := cfg.RecaptchaSiteKey
		if cfg.SkipCaptcha {
			siteKey = ""
		}
		response.JSON(w, http.StatusOK, map[string]string{"site_key": siteKey})
	})

	// Core Web Vitals beacons from the SPA -> Prometheus histograms. Under
	// /api/ so the general rate limiter covers it.
	r.Post("/api/v1/vitals", vitals.Handler(metrics.NewWebVitals(promReg)))

	// Expose Stripe publishable key to frontend (public, non-secret)
	r.Get("/api/v1/config/stripe", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"publishable_key": cfg.StripePublishableKey})
	})

	// Whether this deployment is a public demo — the SPA shows the demo banner
	// (test card + demo login) when true.
	r.Get("/api/v1/config/demo", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]bool{"demo": cfg.DemoMode})
	})

	// Liveness: process is up. Kept dependency-free so restarts don't cascade.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness: verifies the dependencies the app can't serve without.
	// Used by the compose healthcheck and post-deploy validation.
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{}
		healthy := true

		if err := db.Ping(ctx); err != nil {
			checks["database"] = "unreachable: " + err.Error()
			healthy = false
		} else {
			checks["database"] = "ok"
		}

		if amqpConn.IsClosed() {
			checks["rabbitmq"] = "connection closed"
			healthy = false
		} else {
			checks["rabbitmq"] = "ok"
		}

		if rdb == nil {
			checks["redis"] = "disabled"
		} else if err := rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unreachable: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}

		status, code := "ok", http.StatusOK
		if !healthy {
			status, code = "degraded", http.StatusServiceUnavailable
		}
		response.JSON(w, code, map[string]any{"status": status, "checks": checks})
	})

	// Sitemap + robots for search engines. Registered explicitly so they win
	// over the SPA fallback (and the static robots.txt copied from web/public).
	seoHandler := seo.NewHandler(seo.NewService(seo.NewRepository(db), cfg.BaseURL), cfg.BaseURL)
	r.Get("/sitemap.xml", seoHandler.Sitemap)
	r.Get("/robots.txt", seoHandler.Robots)

	// Public contact/support form.
	r.Post("/api/v1/contact", contactHandler.Submit)

	// --- Uploaded product image variants (served from the upload volume) ---
	mediaFS := http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.UploadDir)))
	r.Handle("/media/*", mediaFS)

	// --- Auth routes (no token required; strict rate limit against brute force) ---
	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Middleware)

		r.Post("/api/v1/auth/register", userHandler.Register)
		r.Post("/api/v1/auth/login", authHandler.Login)
		r.Post("/api/v1/auth/refresh", authHandler.Refresh)
		r.Post("/api/v1/auth/forgot-password", authHandler.ForgotPassword)
		r.Post("/api/v1/auth/reset-password", authHandler.ResetPassword)
	})

	// --- OAuth (public) ---
	r.Get("/api/v1/auth/oauth/{provider}", oauthHandler.Redirect)
	r.Get("/api/v1/auth/oauth/{provider}/callback", oauthHandler.Callback)

	// --- Auth routes (token required) ---
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(tokenValidator))
		r.Post("/api/v1/auth/logout", authHandler.Logout)

		// 2FA management
		r.Post("/api/v1/auth/2fa/setup", authHandler.Setup2FA)
		r.Post("/api/v1/auth/2fa/enable", authHandler.Enable2FA)
		r.Post("/api/v1/auth/2fa/disable", authHandler.Disable2FA)
	})

	// --- Public catalog (no auth, but read owner so activity can log) ---
	r.Group(func(r chi.Router) {
		r.Use(mw.OptionalAuth(tokenValidator))
		r.Use(mw.GuestSession(cfg.CookieSecure))

		r.Get("/api/v1/products", productHandler.Search)
		r.Get("/api/v1/products/suggest", productHandler.Suggest)
		r.Get("/api/v1/products/{id}", productHandler.GetByID)

		// Public review list (viewer read from optional auth for vote/own flags).
		r.Get("/api/v1/products/{id}/reviews", reviewHandler.List)
	})
	r.Get("/api/v1/categories", categoryHandler.ListTree)
	r.Get("/api/v1/categories/{slug}", categoryHandler.GetBySlug)
	r.Get("/api/v1/brands", brandHandler.List)
	r.Get("/api/v1/brands/{id}", brandHandler.GetByID)
	r.Get("/api/v1/delivery-options", deliveryHandler.List)

	// --- Cart (works for both logged-in users and guests) ---
	r.Group(func(r chi.Router) {
		r.Use(mw.OptionalAuth(tokenValidator))
		r.Use(mw.GuestSession(cfg.CookieSecure))

		r.Get("/api/v1/cart", cartHandler.GetCart)
		r.Post("/api/v1/cart/items", cartHandler.AddItem)
		r.Put("/api/v1/cart/items/{productId}", cartHandler.UpdateItem)
		r.Delete("/api/v1/cart/items/{productId}", cartHandler.RemoveItem)
		r.Delete("/api/v1/cart", cartHandler.ClearCart)

		// --- Checkout / Orders (same auth surface as cart) ---
		r.Post("/api/v1/checkout", ordersHandler.Checkout)
		r.Get("/api/v1/checkout/prefill", ordersHandler.Prefill)
		r.Get("/api/v1/orders", ordersHandler.List)
		r.Get("/api/v1/orders/{id}", ordersHandler.GetByID)
		r.Post("/api/v1/orders/{id}/cancel", ordersHandler.Cancel)

		// --- Payments (owner-checked) ---
		r.Post("/api/v1/orders/{id}/payment-intent", paymentsHandler.CreateIntent)

		// --- Recommendations (personalised from activity history + cart) ---
		r.Get("/api/v1/recommendations", recommendHandler.Get)
	})

	// --- Stripe webhook (public; signature-verified inside the handler) ---
	r.Post("/api/v1/webhooks/stripe", paymentsHandler.Webhook)

	// --- Cart merge (strict auth; guest cookie read but not auto-issued) ---
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(tokenValidator))

		r.Get("/api/v1/cart/merge-status", cartHandler.GetMergeStatus)
		r.Post("/api/v1/cart/merge", cartHandler.PostMerge)

		// --- Reviews (auth required: purchase-gated submission + helpful votes) ---
		r.Post("/api/v1/products/{id}/reviews", reviewHandler.Create)
		r.Get("/api/v1/products/{id}/reviews/eligibility", reviewHandler.Eligibility)
		r.Post("/api/v1/reviews/{id}/helpful", reviewHandler.Vote)
		r.Delete("/api/v1/reviews/{id}/helpful", reviewHandler.Unvote)

		// --- Saved addresses (auth required; owner-scoped) ---
		r.Get("/api/v1/addresses", addressHandler.List)
		r.Post("/api/v1/addresses", addressHandler.Create)
		r.Put("/api/v1/addresses/{id}", addressHandler.Update)
		r.Delete("/api/v1/addresses/{id}", addressHandler.Delete)
		r.Post("/api/v1/addresses/{id}/default", addressHandler.SetDefault)
	})

	// --- Admin routes (valid token + staff role + enforced 2FA + audited) ---
	// The whole admin surface requires a staff role and 2FA; mutations are
	// audited. Individual sub-groups narrow access by least privilege.
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(tokenValidator))
		r.Use(mw.RequireRole(mw.RoleAdmin, mw.RoleSupport, mw.RoleSales))
		r.Use(mw.RequireStaff2FA(twoFactorChecker))
		r.Use(audit.Middleware(auditRecorder))

		// Probe used by the admin SPA to gate the dashboard: reaching it means
		// the caller is staff with 2FA enabled. Returns the role so the UI can
		// show only the sections that role may use.
		r.Get("/api/v1/admin/me", func(w http.ResponseWriter, req *http.Request) {
			role, _ := req.Context().Value(ctxkey.Role).(string)
			email, _ := req.Context().Value(ctxkey.Email).(string)
			response.JSON(w, http.StatusOK, map[string]string{"role": role, "email": email})
		})

		// Catalog management — admin + sales.
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(mw.RoleAdmin, mw.RoleSales))

			r.Post("/api/v1/admin/products", productHandler.Create)
			r.Post("/api/v1/admin/products/bulk", productHandler.BulkUpload)
			r.Put("/api/v1/admin/products/{id}", productHandler.Update)
			r.Delete("/api/v1/admin/products/{id}", productHandler.Delete)
			r.Post("/api/v1/admin/products/{id}/images", productHandler.AddImage)
			r.Post("/api/v1/admin/products/{id}/images/upload", productHandler.UploadImage)
			r.Delete("/api/v1/admin/products/{id}/images/{imageId}", productHandler.DeleteImage)

			r.Post("/api/v1/admin/categories", categoryHandler.Create)
			r.Put("/api/v1/admin/categories/{id}", categoryHandler.Update)
			r.Delete("/api/v1/admin/categories/{id}", categoryHandler.Delete)
			r.Post("/api/v1/admin/brands", brandHandler.Create)

			r.Get("/api/v1/admin/delivery-options", deliveryHandler.AdminList)
			r.Post("/api/v1/admin/delivery-options", deliveryHandler.Create)
			r.Put("/api/v1/admin/delivery-options/{id}", deliveryHandler.Update)
			r.Delete("/api/v1/admin/delivery-options/{id}", deliveryHandler.Delete)
		})

		// Order management + review moderation — admin + support.
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(mw.RoleAdmin, mw.RoleSupport))

			r.Get("/api/v1/admin/orders", ordersHandler.AdminList)
			r.Get("/api/v1/admin/orders/{id}", ordersHandler.AdminGet)
			r.Patch("/api/v1/admin/orders/{id}/status", ordersHandler.AdminUpdateStatus)
			r.Post("/api/v1/admin/orders/{id}/refund", ordersHandler.AdminRefund)

			r.Get("/api/v1/admin/reviews", reviewHandler.ListModeration)
			r.Patch("/api/v1/admin/reviews/{id}", reviewHandler.SetStatus)
			r.Delete("/api/v1/admin/reviews/{id}", reviewHandler.Delete)
		})

		// User management + audit log — admin only.
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(mw.RoleAdmin))

			r.Get("/api/v1/admin/users", userHandler.AdminList)
			r.Patch("/api/v1/admin/users/{id}/role", userHandler.AdminSetRole)

			r.Get("/api/v1/admin/audit-log", func(w http.ResponseWriter, req *http.Request) {
				page, _ := strconv.Atoi(req.URL.Query().Get("page"))
				pageSize, _ := strconv.Atoi(req.URL.Query().Get("page_size"))
				entries, total, err := auditRecorder.List(req.Context(), page, pageSize)
				if err != nil {
					response.Error(w, http.StatusInternalServerError, "could not load audit log")
					return
				}
				response.JSON(w, http.StatusOK, map[string]interface{}{"entries": entries, "total": total})
			})
		})
	})

	// --- SPA fallback with static-file passthrough ---
	// Any unmatched non-API path first tries to serve a real file from
	// web/dist (Vite copies web/public/ there at build, so /products/foo.jpg
	// resolves the same as it does in `vite dev`). If no file matches, hand
	// the route to React Router via index.html. Registered last so explicit
	// routes still win.
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		if servedStatic(w, req, webDist) {
			return
		}
		http.ServeFile(w, req, webDist+"/index.html")
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
		// Timeouts bound how long a connection can tie up a goroutine — the
		// front line against slow-loris. WriteTimeout sits just above the 30s
		// per-request middleware timeout so legitimate slow requests still finish.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Internal metrics listener. Deliberately a separate server: the port is
	// never published to the host, so /metrics is reachable only from inside
	// the compose network (Prometheus scrapes api:9091).
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", metrics.Handler(promReg))
	metricsSrv := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("metrics listener on %s", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server running on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// --- Optional HTTPS listener (self-signed cert auto-generated) ---
	// Runs alongside HTTP so the Vite dev proxy, Stripe CLI, and healthcheck on
	// the plain port keep working while HTTPS is demonstrable on TLS_PORT.
	var tlsSrv *http.Server
	if cfg.TLSEnabled {
		if err := tlsutil.EnsureSelfSigned(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			log.Fatalf("tls cert: %v", err)
		}
		tlsSrv = &http.Server{
			Addr:              ":" + cfg.TLSPort,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       20 * time.Second,
			WriteTimeout:      35 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			log.Printf("https server running on port %s (self-signed)", cfg.TLSPort)
			if err := tlsSrv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("https server error: %v", err)
			}
		}()
	}

	<-shutdownCtx.Done()

	log.Println("shutting down server...")

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeoutCancel()

	// Stop the consumer first so an in-flight email send can finish before
	// we tear down its connection. Bounded by the same 10s shutdown budget.
	stopConsumer()
	select {
	case <-consumerDone:
	case <-timeoutCtx.Done():
		log.Println("consumer drain timed out")
	}

	if tlsSrv != nil {
		if err := tlsSrv.Shutdown(timeoutCtx); err != nil {
			log.Printf("https server shutdown error: %v", err)
		}
	}

	if err := metricsSrv.Shutdown(timeoutCtx); err != nil {
		log.Printf("metrics server shutdown error: %v", err)
	}

	if err := srv.Shutdown(timeoutCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	// Flush buffered spans last, after all request traffic has drained.
	if err := shutdownTracing(timeoutCtx); err != nil {
		log.Printf("tracing shutdown error: %v", err)
	}

	log.Println("server stopped")
}

// servedStatic returns true if req.URL.Path resolves to a regular file under
// root. Used by the SPA fallback to serve real files (images, icons.svg, the
// Vite-built /assets/*) before handing the route to React Router.
//
// Directory paths and anything that would resolve outside root (via "..") are
// rejected — http.ServeFile would already block traversal, but failing early
// keeps the logic obvious.
// setupLogging installs a slog default handler: JSON in production (for log
// aggregators), human-readable text otherwise. LOG_FORMAT=json|text overrides
// that default — the compose stack sets json so Promtail/Loki can parse dev
// logs too. Level comes from LOG_LEVEL (debug|info|warn|error, default info).
// slog.SetDefault also reroutes the stdlib log package, so existing log.Printf
// calls get level + timestamp structure without touching every call site.
func setupLogging(appEnv, format string) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	useJSON := appEnv == "production"
	switch format {
	case "json":
		useJSON = true
	case "text":
		useJSON = false
	}
	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func servedStatic(w http.ResponseWriter, req *http.Request, root string) bool {
	path := req.URL.Path
	if path == "" || path == "/" {
		return false
	}
	clean := filepath.Clean("/" + path)
	full := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeFile(w, req, full)
	return true
}
