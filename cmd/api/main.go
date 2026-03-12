package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/auth"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("could not load .env file")
	}

	dbURL := os.Getenv("DATABASE_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("could not connect to database")
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal("database is not reachable")
	}

	fmt.Println("connected to database")

	cost, _ := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	jwtSecret := os.Getenv("JWT_SECRET")

	userRepo := user.NewUserRepository(db)
	userService := user.NewService(userRepo, cost)
	userHandler := user.NewHandler(userService)

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewService(userRepo, authRepo, jwtSecret)
	authHandler := auth.NewHandler(authService)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", userHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)

	fmt.Println("server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
