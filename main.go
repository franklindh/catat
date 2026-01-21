package main

import (
	"context"
	"log"

	"github.com/franklindh/catat/api"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/franklindh/catat/docs"
)

// @title           Catat API
// @version         1.0
// @description     API untuk aplikasi pencatatan keuangan pribadi

// @host      localhost:3000
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	conn, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer conn.Close()

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatal("cannot ping database:", err)
	}

	log.Println("Database connected successfully")

	store := db.NewStore(conn)
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	log.Printf("Starting server on %s", config.ServerAddress)
	if err := server.Start(config.ServerAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
