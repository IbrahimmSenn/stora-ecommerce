package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Could not load .env file", err)
		os.Exit(1)
	}
	dbURL := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Println("Could not connect to database", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	fmt.Println("connected to database")
}
