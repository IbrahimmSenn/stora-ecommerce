// main.go — entrypoint. Connects to postgres, wires up all packages, and starts the server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v76"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/activity"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/auth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/brand"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/captcha"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cart"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/category"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/config"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/mailer"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/messaging"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/notifications"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/oauth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/payments"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/recommend"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	log.Println("connected to database")

	// --- Dependency wiring ---

	captchaVerifier := captcha.NewVerifier(cfg.RecaptchaSecretKey, cfg.SkipCaptcha)
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("init encryptor: %v", err)
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

	userRepo := user.NewUserRepository(db)
	userService := user.NewService(userRepo, cfg.BcryptCost, captchaVerifier)
	userHandler := user.NewHandler(userService)

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewService(userRepo, authRepo, cfg.JWTSecret,
		auth.WithMailer(mail),
		auth.WithBaseURL(cfg.BaseURL),
		auth.WithBcryptCost(cfg.BcryptCost),
	)
	authHandler := auth.NewHandler(authService, cfg.CookieSecure)

	brandRepo := brand.NewRepository(db)
	brandService := brand.NewService(brandRepo)
	brandHandler := brand.NewHandler(brandService)

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categoryService)

	activityRepo := activity.NewRepository(db)
	activityService := activity.NewService(activityRepo)

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo)
	productHandler := product.NewHandler(productService, activityService)

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

	// Address verification: Nominatim if a User-Agent is configured (OSM
	// usage policy requires it), otherwise pass through. Logging the choice
	// makes the demo state visible at boot.
	var geocoder orders.Geocoder
	if cfg.NominatimUserAgent != "" {
		geocoder = orders.NewNominatimGeocoder(cfg.NominatimBaseURL, cfg.NominatimUserAgent)
		log.Printf("address verification: nominatim at %s", cfg.NominatimBaseURL)
	} else {
		geocoder = orders.PassthroughGeocoder{}
		log.Println("address verification: disabled (NOMINATIM_USER_AGENT not set)")
	}

	ordersRepo := orders.NewRepository(db)
	ordersService := orders.NewService(ordersRepo, cartService, encryptor, geocoder, refunder, reconciler)
	ordersHandler := orders.NewHandler(ordersService)

	paymentsRepo := payments.NewRepository(db, encryptor)
	paymentsService = payments.NewService(
		paymentsRepo, ordersService, paymentsEventPublisher,
		payments.NewStripeClient(),
		payments.NewStripeRefundClient(),
		cfg.StripeWebhookSecret, cfg.StripePublishableKey,
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

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Frontend (React SPA built into web/dist) ---
	// Static assets and the SPA fallback share one handler below — see the
	// NotFound block at the bottom of the router.
	webDist := "web/dist"

	// Expose reCAPTCHA site key to frontend (public, non-secret)
	r.Get("/api/v1/config/recaptcha", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"site_key": cfg.RecaptchaSiteKey})
	})

	// Expose Stripe publishable key to frontend (public, non-secret)
	r.Get("/api/v1/config/stripe", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"publishable_key": cfg.StripePublishableKey})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Auth routes (no token required) ---
	r.Post("/api/v1/auth/register", userHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/refresh", authHandler.Refresh)
	r.Post("/api/v1/auth/forgot-password", authHandler.ForgotPassword)
	r.Post("/api/v1/auth/reset-password", authHandler.ResetPassword)

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
	})
	r.Get("/api/v1/categories", categoryHandler.ListTree)
	r.Get("/api/v1/categories/{slug}", categoryHandler.GetBySlug)
	r.Get("/api/v1/brands", brandHandler.List)
	r.Get("/api/v1/brands/{id}", brandHandler.GetByID)

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
	})

	// --- Admin routes (valid token + role=admin required) ---
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(tokenValidator))
		r.Use(mw.IsAdmin)

		r.Post("/api/v1/admin/products", productHandler.Create)
		r.Put("/api/v1/admin/products/{id}", productHandler.Update)
		r.Delete("/api/v1/admin/products/{id}", productHandler.Delete)
		r.Post("/api/v1/admin/products/{id}/images", productHandler.AddImage)
		r.Delete("/api/v1/admin/products/{id}/images/{imageId}", productHandler.DeleteImage)

		r.Post("/api/v1/admin/categories", categoryHandler.Create)
		r.Post("/api/v1/admin/brands", brandHandler.Create)
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
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server running on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

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

	if err := srv.Shutdown(timeoutCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
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
