package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"budget-management/internal/config"
	"budget-management/internal/models"
	"budget-management/internal/services"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// TaskProcessor handles background task processing
type TaskProcessor struct {
	server            *asynq.Server
	mux               *asynq.ServeMux
	campaignService   *services.CampaignService
	spendService      *services.SpendService
	daypartingService *services.DaypartingService
	logger            *logrus.Logger
}

// NewTaskProcessor creates a new task processor
func NewTaskProcessor(
	campaignService *services.CampaignService,
	spendService *services.SpendService,
	daypartingService *services.DaypartingService,
	logger *logrus.Logger,
) *TaskProcessor {
	return &TaskProcessor{
		campaignService:   campaignService,
		spendService:      spendService,
		daypartingService: daypartingService,
		logger:            logger,
	}
}

// Initialize sets up the task processor with Redis configuration
func (p *TaskProcessor) Initialize(cfg config.RedisConfig, tasksCfg config.TasksConfig) {
	p.server = asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.GetRedisAddr(),
			Password: cfg.Password,
			DB:       cfg.DB,
		},
		asynq.Config{
			Concurrency: tasksCfg.Concurrency,
			Queues:      tasksCfg.Queues,
			Logger:      p.logger,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				p.logger.Errorf("Task failed: %s, Error: %v", task.Type(), err)
			}),
		},
	)

	p.mux = asynq.NewServeMux()
	p.registerHandlers()
}

// registerHandlers registers task handlers
func (p *TaskProcessor) registerHandlers() {
	p.mux.HandleFunc(TaskRecordSpend, p.handleRecordSpend)
	p.mux.HandleFunc(TaskCheckBudget, p.handleCheckBudget)
	p.mux.HandleFunc(TaskDailyReset, p.handleDailyReset)
	p.mux.HandleFunc(TaskMonthlyReset, p.handleMonthlyReset)
	p.mux.HandleFunc(TaskDaypartingCheck, p.handleDaypartingCheck)
	p.mux.HandleFunc(TaskPeriodicBudgetCheck, p.handlePeriodicBudgetCheck)
}

// Start starts the task processor
func (p *TaskProcessor) Start(ctx context.Context) error {
	p.logger.Info("Starting task processor...")
	return p.server.Run(p.mux)
}

// Stop stops the task processor
func (p *TaskProcessor) Stop() {
	p.logger.Info("Stopping task processor...")
	p.server.Stop()
	p.server.Shutdown()
}

// handleRecordSpend handles spend recording tasks
func (p *TaskProcessor) handleRecordSpend(ctx context.Context, task *asynq.Task) error {
	var payload models.TaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	campaignID, err := uuid.Parse(payload.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign ID: %w", err)
	}

	amount, err := decimal.NewFromString(payload.Amount)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	var spendDate *time.Time
	if payload.SpendDate != "" {
		date, err := time.Parse("2006-01-02", payload.SpendDate)
		if err != nil {
			return fmt.Errorf("invalid spend date: %w", err)
		}
		spendDate = &date
	}

	// Record the spend
	if err := p.spendService.RecordSpend(ctx, campaignID, amount, spendDate); err != nil {
		return fmt.Errorf("failed to record spend: %w", err)
	}

	// Trigger budget check
	if err := p.campaignService.CheckBudgetLimits(ctx, campaignID); err != nil {
		p.logger.Errorf("Failed to check budget limits after spend recording: %v", err)
		// Don't fail the task for budget check errors
	}

	p.logger.Infof("Successfully recorded spend of %s for campaign %s", amount, campaignID)
	return nil
}

// handleCheckBudget handles budget checking tasks
func (p *TaskProcessor) handleCheckBudget(ctx context.Context, task *asynq.Task) error {
	var payload models.TaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	campaignID, err := uuid.Parse(payload.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign ID: %w", err)
	}

	if err := p.campaignService.CheckBudgetLimits(ctx, campaignID); err != nil {
		return fmt.Errorf("failed to check budget limits: %w", err)
	}

	p.logger.Debugf("Successfully checked budget limits for campaign %s", campaignID)
	return nil
}

// handleDailyReset handles daily reset tasks
func (p *TaskProcessor) handleDailyReset(ctx context.Context, task *asynq.Task) error {
	reactivatedCount, err := p.campaignService.DailyReset(ctx)
	if err != nil {
		return fmt.Errorf("failed to perform daily reset: %w", err)
	}

	p.logger.Infof("Daily reset completed: reactivated %d campaigns", reactivatedCount)
	return nil
}

// handleMonthlyReset handles monthly reset tasks
func (p *TaskProcessor) handleMonthlyReset(ctx context.Context, task *asynq.Task) error {
	reactivatedCount, err := p.campaignService.MonthlyReset(ctx)
	if err != nil {
		return fmt.Errorf("failed to perform monthly reset: %w", err)
	}

	p.logger.Infof("Monthly reset completed: reactivated %d campaigns", reactivatedCount)
	return nil
}

// handleDaypartingCheck handles dayparting check tasks
func (p *TaskProcessor) handleDaypartingCheck(ctx context.Context, task *asynq.Task) error {
	activatedCount, deactivatedCount, err := p.daypartingService.CheckDayparting(ctx)
	if err != nil {
		return fmt.Errorf("failed to check dayparting: %w", err)
	}

	p.logger.Debugf("Dayparting check completed: activated %d, deactivated %d campaigns", 
		activatedCount, deactivatedCount)
	return nil
}

// handlePeriodicBudgetCheck handles periodic budget check tasks
func (p *TaskProcessor) handlePeriodicBudgetCheck(ctx context.Context, task *asynq.Task) error {
	checkedCount, err := p.campaignService.PeriodicBudgetCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to perform periodic budget check: %w", err)
	}

	p.logger.Infof("Periodic budget check completed: checked %d campaigns", checkedCount)
	return nil
}

// GetServerInfo returns information about the task processor server
func (p *TaskProcessor) GetServerInfo(ctx context.Context) (*asynq.ServerInfo, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: p.server.(*asynq.Server).Options().Addr,
	})
	defer inspector.Close()

	servers, err := inspector.GetServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	if len(servers) > 0 {
		return servers[0], nil
	}

	return nil, fmt.Errorf("no servers found")
}

// GetTaskInfo returns information about active and pending tasks
func (p *TaskProcessor) GetTaskInfo(ctx context.Context, queueName string) (*asynq.QueueInfo, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: p.server.(*asynq.Server).Options().Addr,
	})
	defer inspector.Close()

	return inspector.GetQueueInfo(queueName)
}

// RetryFailedTasks retries all failed tasks in a queue
func (p *TaskProcessor) RetryFailedTasks(ctx context.Context, queueName string) (int, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: p.server.(*asynq.Server).Options().Addr,
	})
	defer inspector.Close()

	tasks, err := inspector.ListDeadTasks(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to list dead tasks: %w", err)
	}

	retryCount := 0
	for _, task := range tasks {
		if err := inspector.RetryTask(queueName, task.ID); err != nil {
			p.logger.Errorf("Failed to retry task %s: %v", task.ID, err)
			continue
		}
		retryCount++
	}

	p.logger.Infof("Retried %d failed tasks in queue %s", retryCount, queueName)
	return retryCount, nil
}

// CancelTask cancels a specific task
func (p *TaskProcessor) CancelTask(ctx context.Context, queueName, taskID string) error {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: p.server.(*asynq.Server).Options().Addr,
	})
	defer inspector.Close()

	if err := inspector.CancelProcessing(taskID); err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	p.logger.Infof("Cancelled task %s in queue %s", taskID, queueName)
	return nil
}

// GetTaskStats returns task processing statistics
func (p *TaskProcessor) GetTaskStats(ctx context.Context) (map[string]interface{}, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: p.server.(*asynq.Server).Options().Addr,
	})
	defer inspector.Close()

	stats := make(map[string]interface{})

	// Get queue information
	queues := []string{"critical", "default", "low"}
	for _, queueName := range queues {
		queueInfo, err := inspector.GetQueueInfo(queueName)
		if err != nil {
			p.logger.Errorf("Failed to get info for queue %s: %v", queueName, err)
			continue
		}
		stats[queueName] = queueInfo
	}

	// Get server information
	servers, err := inspector.GetServers()
	if err != nil {
		p.logger.Errorf("Failed to get server info: %v", err)
	} else {
		stats["servers"] = servers
	}

	return stats, nil
}