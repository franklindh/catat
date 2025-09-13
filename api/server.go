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
	Store  db.Store
	Router *gin.Engine
}

func NewServer(dbPool *pgxpool.Pool) *Server {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Error loading .env file")
	}

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	Router := gin.New()
	Router.Use(gin.Logger())
	Router.Use(gin.Recovery())

	// Create Store from db pool
	Store := db.NewStore(dbPool)

	server := &Server{
		Store:  Store,
		Router: Router,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Catat API is running",
		})
	})

	s.Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Catat API",
			"version": "1.0.0",
		})
	})

	// Account routes
	s.Router.POST("/accounts", s.createAccount)
	s.Router.GET("/accounts/:id", s.getAccount)
	s.Router.GET("/accounts", s.listAccounts)
	s.Router.PUT("/accounts", s.updateAccount)
	s.Router.DELETE("/accounts/:id", s.deleteAccount)

	// User routes
	s.Router.POST("/users", s.createUser)
	s.Router.GET("/users/:id", s.getUserByID)
	s.Router.GET("/users", s.getUserByEmail)
	s.Router.GET("/users/list", s.listUsers)
	s.Router.PUT("/users", s.updateUser)
	s.Router.DELETE("/users/:id", s.deleteUser)

	// Category routes
	s.Router.POST("/categories", s.createCategory)
	s.Router.GET("/categories/:id", s.getCategory)
	s.Router.GET("/categories", s.listCategories)
	s.Router.PUT("/categories", s.updateCategory)
	s.Router.DELETE("/categories/:id", s.deleteCategory)

	// Transaction routes
	s.Router.POST("/transactions", s.createTransaction)
	s.Router.GET("/transactions/:id", s.getTransaction)
	s.Router.GET("/transactions", s.listTransactions)
	s.Router.GET("/transactions/account", s.listTransactionsByAccount)
	s.Router.GET("/transactions/date-range", s.listTransactionsByDateRange)
	s.Router.PUT("/transactions", s.updateTransaction)
	s.Router.DELETE("/transactions/:id", s.deleteTransaction)

	// Receipt routes
	s.Router.POST("/receipts", s.createReceipt)
	s.Router.GET("/receipts/transaction/:transaction_id", s.getReceiptByTransactionID)
	s.Router.DELETE("/receipts/:id", s.deleteReceipt)
}

func (s *Server) Start(address string) error {
	server := &http.Server{
		Addr:    address,
		Handler: s.Router,
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
	return s.Store
}
