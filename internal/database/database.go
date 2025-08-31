package database

import (
	"fmt"
	"time"

	"budget-management/internal/config"
	"budget-management/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes database connection
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.GetDSN())
	case "sqlite":
		dialector = sqlite.Open(cfg.GetDSN())
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	// Auto-migrate all models
	err := db.AutoMigrate(
		&models.Brand{},
		&models.Campaign{},
		&models.DaypartingSchedule{},
		&models.SpendRecord{},
		&models.DailySpendAggregate{},
		&models.MonthlySpendAggregate{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create indexes
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates additional database indexes
func createIndexes(db *gorm.DB) error {
	// Campaign indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_campaigns_active_budget ON campaigns(is_active, is_paused_by_budget)").Error; err != nil {
		return err
	}

	// Spend record indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_spend_records_campaign_date ON spend_records(campaign_id, spend_date)").Error; err != nil {
		return err
	}

	// Daily aggregate indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_daily_aggregates_date_campaign ON daily_spend_aggregates(date, campaign_id)").Error; err != nil {
		return err
	}

	// Monthly aggregate indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_monthly_aggregates_year_month_campaign ON monthly_spend_aggregates(year, month, campaign_id)").Error; err != nil {
		return err
	}

	// Dayparting schedule indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_dayparting_campaign_day_active ON dayparting_schedules(campaign_id, day_of_week, is_active)").Error; err != nil {
		return err
	}

	return nil
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
		MinIdleConns: 5,
	})
}

// HealthCheck checks database connectivity
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// RedisHealthCheck checks Redis connectivity
func RedisHealthCheck(client *redis.Client) error {
	return client.Ping(client.Context()).Err()
}