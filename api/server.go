package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Server struct {
	store  db.Store
	router *gin.Engine
}

func NewServer(dbPool *pgxpool.Pool) *Server {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Error loading .env file")
	}

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Create store from db pool
	store := db.NewStore(dbPool)

	server := &Server{
		store:  store,
		router: router,
	}

	server.setupRoutes()
	return server
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

	// Account routes
	s.router.POST("/accounts", s.createAccount)
	s.router.GET("/accounts/:id", s.getAccount)
	s.router.GET("/accounts", s.listAccounts)
	s.router.PUT("/accounts", s.updateAccount)
	s.router.DELETE("/accounts/:id", s.deleteAccount)

	// User routes
	s.router.POST("/users", s.createUser)
	s.router.GET("/users/:id", s.getUserByID)
	s.router.GET("/users", s.getUserByEmail)
	s.router.GET("/users/list", s.listUsers)
	s.router.PUT("/users", s.updateUser)
	s.router.DELETE("/users/:id", s.deleteUser)

	// Category routes
	s.router.POST("/categories", s.createCategory)
	s.router.GET("/categories/:id", s.getCategory)
	s.router.GET("/categories", s.listCategories)
	s.router.PUT("/categories", s.updateCategory)
	s.router.DELETE("/categories/:id", s.deleteCategory)

	// Transaction routes
	s.router.POST("/transactions", s.createTransaction)
	s.router.GET("/transactions/:id", s.getTransaction)
	s.router.GET("/transactions", s.listTransactions)
	s.router.GET("/transactions/account", s.listTransactionsByAccount)
	s.router.GET("/transactions/date-range", s.listTransactionsByDateRange)
	s.router.PUT("/transactions", s.updateTransaction)
	s.router.DELETE("/transactions/:id", s.deleteTransaction)

	// Receipt routes
	s.router.POST("/receipts", s.createReceipt)
	s.router.GET("/receipts/transaction/:transaction_id", s.getReceiptByTransactionID)
	s.router.DELETE("/receipts/:id", s.deleteReceipt)
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

// Getter methods (jika masih dibutuhkan untuk backward compatibility)
func (s *Server) GetStore() db.Store {
	return s.store
}
