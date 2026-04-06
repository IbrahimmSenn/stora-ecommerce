// main.go — entrypoint. Connects to postgres, wires up all packages, and starts the server.
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/auth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/brand"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/captcha"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/category"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/config"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/mailer"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/oauth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
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

	userRepo := user.NewUserRepository(db)
	userService := user.NewService(userRepo, cfg.BcryptCost, captchaVerifier)
	userHandler := user.NewHandler(userService)

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewService(userRepo, authRepo, cfg.JWTSecret,
		auth.WithMailer(mail),
		auth.WithBaseURL(cfg.BaseURL),
		auth.WithBcryptCost(cfg.BcryptCost),
	)
	authHandler := auth.NewHandler(authService)

	brandRepo := brand.NewRepository(db)
	brandService := brand.NewService(brandRepo)
	brandHandler := brand.NewHandler(brandService)

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categoryService)

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo)
	productHandler := product.NewHandler(productService)

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

	oauthHandler := oauth.NewHandler(oauthService, providers, cfg.BaseURL)

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

	// --- Static frontend (test panel served at /) ---
	staticFS := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", staticFS))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	// Expose reCAPTCHA site key to frontend (public, non-secret)
	r.Get("/api/v1/config/recaptcha", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"site_key": cfg.RecaptchaSiteKey})
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

	// --- Public catalog (no auth) ---
	r.Get("/api/v1/products", productHandler.Search)
	r.Get("/api/v1/products/suggest", productHandler.Suggest)
	r.Get("/api/v1/products/{id}", productHandler.GetByID)
	r.Get("/api/v1/categories", categoryHandler.ListTree)
	r.Get("/api/v1/categories/{slug}", categoryHandler.GetBySlug)
	r.Get("/api/v1/brands", brandHandler.List)
	r.Get("/api/v1/brands/{id}", brandHandler.GetByID)

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

	if err := srv.Shutdown(timeoutCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	log.Println("server stopped")
}
