package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"budget-management/internal/config"
	"budget-management/internal/database"
	"budget-management/internal/handlers"
	"budget-management/internal/middleware"
	"budget-management/internal/repository"
	"budget-management/internal/services"
	"budget-management/internal/tasks"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.Level(cfg.LogLevel))
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database schema
	if err := database.Migrate(db); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize Redis client
	redisClient := database.NewRedisClient(cfg.Redis)

	// Initialize repositories
	brandRepo := repository.NewBrandRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	spendRepo := repository.NewSpendRepository(db)
	aggregateRepo := repository.NewAggregateRepository(db)
	daypartingRepo := repository.NewDaypartingRepository(db)

	// Initialize services
	lockService := services.NewLockService(redisClient)
	campaignService := services.NewCampaignService(campaignRepo, spendRepo, aggregateRepo, lockService, logger)
	spendService := services.NewSpendService(spendRepo, aggregateRepo, campaignService, lockService, logger)
	daypartingService := services.NewDaypartingService(daypartingRepo, campaignRepo, logger)

	// Initialize task queue
	taskQueue := tasks.NewTaskQueue(cfg.Redis, logger)
	taskProcessor := tasks.NewTaskProcessor(
		campaignService,
		spendService, 
		daypartingService,
		logger,
	)

	// Start task processor
	go func() {
		if err := taskProcessor.Start(context.Background()); err != nil {
			logger.Errorf("Task processor error: %v", err)
		}
	}()

	// Initialize handlers
	brandHandler := handlers.NewBrandHandler(brandRepo, logger)
	campaignHandler := handlers.NewCampaignHandler(campaignService, logger)
	spendHandler := handlers.NewSpendHandler(spendService, taskQueue, logger)
	daypartingHandler := handlers.NewDaypartingHandler(daypartingService, logger)
	healthHandler := handlers.NewHealthHandler(db, redisClient, logger)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))

	// Health check endpoints
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Brand routes
		brands := v1.Group("/brands")
		{
			brands.GET("", brandHandler.List)
			brands.GET("/:id", brandHandler.Get)
			brands.POST("", brandHandler.Create)
			brands.PUT("/:id", brandHandler.Update)
			brands.DELETE("/:id", brandHandler.Delete)
		}

		// Campaign routes
		campaigns := v1.Group("/campaigns")
		{
			campaigns.GET("", campaignHandler.List)
			campaigns.GET("/:id", campaignHandler.Get)
			campaigns.POST("", campaignHandler.Create)
			campaigns.PUT("/:id", campaignHandler.Update)
			campaigns.DELETE("/:id", campaignHandler.Delete)
			campaigns.POST("/:id/pause", campaignHandler.Pause)
			campaigns.POST("/:id/activate", campaignHandler.Activate)
			campaigns.GET("/:id/spend", campaignHandler.GetSpend)
		}

		// Spend routes
		spend := v1.Group("/spend")
		{
			spend.POST("", spendHandler.Record)
			spend.GET("/campaign/:id", spendHandler.GetCampaignSpend)
			spend.GET("/campaign/:id/daily", spendHandler.GetDailySpend)
			spend.GET("/campaign/:id/monthly", spendHandler.GetMonthlySpend)
		}

		// Dayparting routes
		dayparting := v1.Group("/dayparting")
		{
			dayparting.GET("/campaign/:id", daypartingHandler.List)
			dayparting.POST("/campaign/:id", daypartingHandler.Create)
			dayparting.PUT("/:id", daypartingHandler.Update)
			dayparting.DELETE("/:id", daypartingHandler.Delete)
		}
	}

	// Setup HTTP server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown task processor
	taskProcessor.Stop()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}