package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"budget-management/internal/config"
	"budget-management/internal/database"
	"budget-management/internal/repository"
	"budget-management/internal/services"
	"budget-management/internal/tasks"

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

	// Initialize Redis client
	redisClient := database.NewRedisClient(cfg.Redis)

	// Initialize repositories
	campaignRepo := repository.NewCampaignRepository(db)
	spendRepo := repository.NewSpendRepository(db)
	aggregateRepo := repository.NewAggregateRepository(db)
	daypartingRepo := repository.NewDaypartingRepository(db)

	// Initialize services
	lockService := services.NewLockService(redisClient)
	campaignService := services.NewCampaignService(campaignRepo, spendRepo, aggregateRepo, lockService, logger)
	spendService := services.NewSpendService(spendRepo, aggregateRepo, campaignService, lockService, logger)
	daypartingService := services.NewDaypartingService(daypartingRepo, campaignRepo, logger)

	// Initialize task processor
	taskProcessor := tasks.NewTaskProcessor(
		campaignService,
		spendService,
		daypartingService,
		logger,
	)

	// Initialize scheduler for periodic tasks
	scheduler := tasks.NewScheduler(
		campaignService,
		spendService,
		daypartingService,
		logger,
	)

	// Start scheduler
	go func() {
		logger.Info("Starting task scheduler...")
		scheduler.Start()
	}()

	// Start task processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Info("Starting task processor...")
		if err := taskProcessor.Start(ctx); err != nil {
			logger.Errorf("Task processor error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down worker...")

	// Stop scheduler
	scheduler.Stop()

	// Stop task processor
	taskProcessor.Stop()

	logger.Info("Worker exited")
}