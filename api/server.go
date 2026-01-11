package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Error loading .env file")
	}

	gin.SetMode(gin.DebugMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	Router := gin.New()
	Router.Use(gin.Logger())
	Router.Use(gin.Recovery())

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		router:     Router,
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {

	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Catat API is running",
		})
	})

	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Catat API",
			"version": "1.0.0",
		})
	})

	s.router.GET("/users/google/login", s.googleOAuthLogin)
	s.router.GET("/users/google/callback", s.googleOAuthCallback)
	s.router.POST("/users/google/callback", s.googleOAuthCallback)

	s.router.GET("/users/:id", s.getUser)

	authRoutes := s.router.Group("/").Use(authMiddleware(s.tokenMaker))
	{
		authRoutes.POST("/categories", s.createCategory)
		authRoutes.GET("/categories", s.getCategory)
		authRoutes.GET("/categories/:id", s.getCategoryByID)
		authRoutes.PUT("/categories/:id", s.updateCategory)
		authRoutes.DELETE("/categories/:id", s.deleteCategory)

		authRoutes.POST("/transactions", s.createTransaction)
		authRoutes.GET("/transactions/:id", s.getTransactions)
		authRoutes.GET("/transaction", s.getTransaction)

		authRoutes.PUT("/transactions/:id", s.updateTransaction)
		authRoutes.DELETE("/transactions/:id", s.deleteTransaction)

		authRoutes.GET("/user", s.getUser)
		authRoutes.PUT("/user", s.updateUser)
		authRoutes.DELETE("/user", s.deleteUser)
	}
}

func (s *Server) Start(address string) error {
	server := &http.Server{
		Addr:    address,
		Handler: s.router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("Server is running on %s", address)
	log.Println("Press Ctrl+C to stop")

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exited gracefully")
	return nil
}

func (s *Server) GetStore() db.Store {
	return s.store
}
