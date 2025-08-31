package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"budget-management/internal/config"
	"budget-management/internal/models"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

// Task types
const (
	TaskRecordSpend       = "spend:record"
	TaskCheckBudget       = "budget:check"
	TaskDailyReset        = "reset:daily"
	TaskMonthlyReset      = "reset:monthly"
	TaskDaypartingCheck   = "dayparting:check"
	TaskPeriodicBudgetCheck = "budget:periodic_check"
)

// TaskQueue handles background task queuing
type TaskQueue struct {
	client *asynq.Client
	logger *logrus.Logger
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(cfg config.RedisConfig, logger *logrus.Logger) *TaskQueue {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &TaskQueue{
		client: client,
		logger: logger,
	}
}

// Close closes the task queue client
func (q *TaskQueue) Close() error {
	return q.client.Close()
}

// EnqueueRecordSpend queues a spend recording task
func (q *TaskQueue) EnqueueRecordSpend(campaignID, amount, spendDate string) error {
	payload := models.TaskPayload{
		CampaignID: campaignID,
		Amount:     amount,
		SpendDate:  spendDate,
		TaskType:   TaskRecordSpend,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskRecordSpend, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("critical"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Infof("Enqueued spend recording task: %s", info.ID)
	return nil
}

// EnqueueBudgetCheck queues a budget check task
func (q *TaskQueue) EnqueueBudgetCheck(campaignID string) error {
	payload := models.TaskPayload{
		CampaignID: campaignID,
		TaskType:   TaskCheckBudget,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskCheckBudget, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Debugf("Enqueued budget check task: %s", info.ID)
	return nil
}

// EnqueueDailyReset queues a daily reset task
func (q *TaskQueue) EnqueueDailyReset() error {
	payload := models.TaskPayload{
		TaskType: TaskDailyReset,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskDailyReset, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("critical"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Infof("Enqueued daily reset task: %s", info.ID)
	return nil
}

// EnqueueMonthlyReset queues a monthly reset task
func (q *TaskQueue) EnqueueMonthlyReset() error {
	payload := models.TaskPayload{
		TaskType: TaskMonthlyReset,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskMonthlyReset, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("critical"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Infof("Enqueued monthly reset task: %s", info.ID)
	return nil
}

// EnqueueDaypartingCheck queues a dayparting check task
func (q *TaskQueue) EnqueueDaypartingCheck() error {
	payload := models.TaskPayload{
		TaskType: TaskDaypartingCheck,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskDaypartingCheck, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Debugf("Enqueued dayparting check task: %s", info.ID)
	return nil
}

// EnqueuePeriodicBudgetCheck queues a periodic budget check task
func (q *TaskQueue) EnqueuePeriodicBudgetCheck() error {
	payload := models.TaskPayload{
		TaskType: TaskPeriodicBudgetCheck,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskPeriodicBudgetCheck, data)
	
	info, err := q.client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Debugf("Enqueued periodic budget check task: %s", info.ID)
	return nil
}

// EnqueueDelayedTask queues a task to be executed at a specific time
func (q *TaskQueue) EnqueueDelayedTask(taskType string, payload models.TaskPayload, executeAt time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, data)
	
	info, err := q.client.Enqueue(task, asynq.ProcessAt(executeAt))
	if err != nil {
		return fmt.Errorf("failed to enqueue delayed task: %w", err)
	}

	q.logger.Infof("Enqueued delayed task %s: %s (execute at: %s)", taskType, info.ID, executeAt)
	return nil
}

// EnqueueRecurringTask queues a task with retry options
func (q *TaskQueue) EnqueueRecurringTask(taskType string, payload models.TaskPayload, maxRetry int) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, data)
	
	info, err := q.client.Enqueue(task, asynq.MaxRetry(maxRetry))
	if err != nil {
		return fmt.Errorf("failed to enqueue recurring task: %w", err)
	}

	q.logger.Infof("Enqueued recurring task %s: %s (max retry: %d)", taskType, info.ID, maxRetry)
	return nil
}

// GetQueueInfo returns information about the task queues
func (q *TaskQueue) GetQueueInfo(ctx context.Context) (map[string]*asynq.QueueInfo, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: q.client.(*asynq.Client).Options().Addr,
	})
	defer inspector.Close()

	queues := []string{"critical", "default", "low"}
	queueInfo := make(map[string]*asynq.QueueInfo)

	for _, queueName := range queues {
		info, err := inspector.GetQueueInfo(queueName)
		if err != nil {
			q.logger.Errorf("Failed to get info for queue %s: %v", queueName, err)
			continue
		}
		queueInfo[queueName] = info
	}

	return queueInfo, nil
}

// PurgeQueue removes all tasks from a specific queue
func (q *TaskQueue) PurgeQueue(ctx context.Context, queueName string) (int, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: q.client.(*asynq.Client).Options().Addr,
	})
	defer inspector.Close()

	n, err := inspector.DeleteAllPendingTasks(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to purge queue %s: %w", queueName, err)
	}

	q.logger.Infof("Purged %d tasks from queue %s", n, queueName)
	return n, nil
}