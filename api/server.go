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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	Router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	rateLimiter := NewRateLimiter(100, time.Minute)
	Router.Use(RateLimitMiddleware(rateLimiter))

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

	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

		authRoutes.GET("/dashboard", s.getDashboard)
		authRoutes.GET("/dashboard/balance", s.getTotalBalance)
		authRoutes.GET("/dashboard/expenses-by-category", s.getExpenseByCategory)
		authRoutes.GET("/dashboard/daily-expense-trend", s.getDailyExpenseTrend)
	}

	adminRoutes := s.router.Group("/admin").Use(authMiddleware(s.tokenMaker), requireRole(RoleAdmin))
	{
		adminRoutes.GET("/categories", s.getCategory)
		adminRoutes.GET("/users", s.listUsers)
		adminRoutes.GET("/users/:id", s.getUserByID)
		adminRoutes.PUT("/users/:id/role", s.updateUserRole)
		adminRoutes.DELETE("/users/:id", s.deleteUserByAdmin)
		adminRoutes.GET("/stats", s.getAdminStats)
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
