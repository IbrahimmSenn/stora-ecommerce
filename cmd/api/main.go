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
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/auth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/config"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
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

	userRepo := user.NewUserRepository(db)
	userService := user.NewService(userRepo, cfg.BcryptCost)
	userHandler := user.NewHandler(userService)

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewService(userRepo, authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/api/v1/auth/register", userHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/refresh", authHandler.Refresh)

	// Protected routes — require valid access token
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(func(tokenString string) (*mw.TokenClaims, error) {
			claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
			if err != nil {
				return nil, err
			}
			return &mw.TokenClaims{
				UserID: claims.UserID,
				Email:  claims.Email,
				Role:   claims.Role,
			}, nil
		}))
		r.Post("/api/v1/auth/logout", authHandler.Logout)
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
