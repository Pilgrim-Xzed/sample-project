package tasks

import (
	"context"
	"time"

	"budget-management/internal/services"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// Scheduler handles periodic task scheduling
type Scheduler struct {
	cron              *cron.Cron
	campaignService   *services.CampaignService
	spendService      *services.SpendService
	daypartingService *services.DaypartingService
	logger            *logrus.Logger
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewScheduler creates a new task scheduler
func NewScheduler(
	campaignService *services.CampaignService,
	spendService *services.SpendService,
	daypartingService *services.DaypartingService,
	logger *logrus.Logger,
) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:              cron.New(cron.WithSeconds()),
		campaignService:   campaignService,
		spendService:      spendService,
		daypartingService: daypartingService,
		logger:            logger,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start starts the scheduler and registers all periodic tasks
func (s *Scheduler) Start() {
	s.logger.Info("Starting task scheduler...")

	// Daily reset task - runs at midnight daily
	_, err := s.cron.AddFunc("0 0 0 * * *", func() {
		s.runWithContext("daily_reset", func(ctx context.Context) error {
			_, err := s.campaignService.DailyReset(ctx)
			return err
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule daily reset task: %v", err)
	}

	// Monthly reset task - runs at midnight on the first day of each month
	_, err = s.cron.AddFunc("0 0 0 1 * *", func() {
		s.runWithContext("monthly_reset", func(ctx context.Context) error {
			_, err := s.campaignService.MonthlyReset(ctx)
			return err
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule monthly reset task: %v", err)
	}

	// Dayparting check task - runs every minute
	_, err = s.cron.AddFunc("0 * * * * *", func() {
		s.runWithContext("dayparting_check", func(ctx context.Context) error {
			_, _, err := s.daypartingService.CheckDayparting(ctx)
			return err
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule dayparting check task: %v", err)
	}

	// Periodic budget check task - runs every 5 minutes
	_, err = s.cron.AddFunc("0 */5 * * * *", func() {
		s.runWithContext("periodic_budget_check", func(ctx context.Context) error {
			_, err := s.campaignService.PeriodicBudgetCheck(ctx)
			return err
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule periodic budget check task: %v", err)
	}

	// Health check task - runs every 30 seconds
	_, err = s.cron.AddFunc("*/30 * * * * *", func() {
		s.runWithContext("health_check", func(ctx context.Context) error {
			return s.performHealthCheck(ctx)
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule health check task: %v", err)
	}

	// Cleanup old data task - runs daily at 2 AM
	_, err = s.cron.AddFunc("0 0 2 * * *", func() {
		s.runWithContext("cleanup_old_data", func(ctx context.Context) error {
			return s.cleanupOldData(ctx)
		})
	})
	if err != nil {
		s.logger.Errorf("Failed to schedule cleanup task: %v", err)
	}

	s.cron.Start()
	s.logger.Info("Task scheduler started successfully")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping task scheduler...")
	s.cancel()
	
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		s.logger.Info("Task scheduler stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Task scheduler stop timeout")
	}
}

// runWithContext runs a task function with context and error handling
func (s *Scheduler) runWithContext(taskName string, fn func(context.Context) error) {
	start := time.Now()
	s.logger.Debugf("Starting scheduled task: %s", taskName)

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()

	if err := fn(ctx); err != nil {
		s.logger.Errorf("Scheduled task %s failed: %v", taskName, err)
	} else {
		duration := time.Since(start)
		s.logger.Infof("Scheduled task %s completed successfully in %v", taskName, duration)
	}
}

// performHealthCheck performs a basic health check
func (s *Scheduler) performHealthCheck(ctx context.Context) error {
	s.logger.Debug("Performing health check...")
	
	// This is a placeholder for health check logic
	// In a real implementation, you might check:
	// - Database connectivity
	// - Redis connectivity  
	// - External service availability
	// - System resources
	
	return nil
}

// cleanupOldData performs cleanup of old data
func (s *Scheduler) cleanupOldData(ctx context.Context) error {
	s.logger.Info("Starting cleanup of old data...")

	// This is a placeholder for cleanup logic
	// In a real implementation, you might:
	// - Archive old spend records
	// - Clean up expired locks
	// - Remove old log entries
	// - Compress historical data

	s.logger.Info("Cleanup of old data completed")
	return nil
}

// AddCustomTask adds a custom scheduled task
func (s *Scheduler) AddCustomTask(schedule string, taskName string, fn func(context.Context) error) error {
	_, err := s.cron.AddFunc(schedule, func() {
		s.runWithContext(taskName, fn)
	})
	
	if err != nil {
		return err
	}
	
	s.logger.Infof("Added custom scheduled task: %s with schedule: %s", taskName, schedule)
	return nil
}

// RemoveTask removes a scheduled task by ID
func (s *Scheduler) RemoveTask(entryID cron.EntryID) {
	s.cron.Remove(entryID)
	s.logger.Infof("Removed scheduled task with ID: %d", entryID)
}

// GetScheduledTasks returns information about all scheduled tasks
func (s *Scheduler) GetScheduledTasks() []cron.Entry {
	return s.cron.Entries()
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	entries := s.cron.Entries()
	return len(entries) > 0
}

// GetNextRun returns the next run time for all scheduled tasks
func (s *Scheduler) GetNextRun() map[cron.EntryID]time.Time {
	entries := s.cron.Entries()
	nextRuns := make(map[cron.EntryID]time.Time)
	
	for _, entry := range entries {
		nextRuns[entry.ID] = entry.Next
	}
	
	return nextRuns
}

// TriggerTask manually triggers a specific task type
func (s *Scheduler) TriggerTask(ctx context.Context, taskType string) error {
	switch taskType {
	case "daily_reset":
		_, err := s.campaignService.DailyReset(ctx)
		return err
	case "monthly_reset":
		_, err := s.campaignService.MonthlyReset(ctx)
		return err
	case "dayparting_check":
		_, _, err := s.daypartingService.CheckDayparting(ctx)
		return err
	case "periodic_budget_check":
		_, err := s.campaignService.PeriodicBudgetCheck(ctx)
		return err
	case "health_check":
		return s.performHealthCheck(ctx)
	case "cleanup_old_data":
		return s.cleanupOldData(ctx)
	default:
		return fmt.Errorf("unknown task type: %s", taskType)
	}
}

// GetTaskStats returns statistics about scheduled tasks
func (s *Scheduler) GetTaskStats() map[string]interface{} {
	entries := s.cron.Entries()
	
	stats := map[string]interface{}{
		"total_tasks":    len(entries),
		"scheduler_running": s.IsRunning(),
		"next_runs":      s.GetNextRun(),
	}
	
	return stats
}