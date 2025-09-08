package main

import (
	"context"
	"log"
	"os"

	server "github.com/franklindh/catat/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using default environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:password@localhost:5432/catat_db?sslmode=disable"
		log.Println("Using default database URL")
	}

	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer conn.Close()

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatal("cannot ping database:", err)
	}

	log.Println("Database connected successfully")

	server := server.NewServer(conn)

	address := os.Getenv("SERVER_ADDR")
	if address == "" {
		address = ":3000"
	}

	if err := server.Start(address); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
