package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/scripts"
)

// TaskStatus represents the current state of a scheduled task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusComplete  TaskStatus = "complete"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents a scheduled task
type Task struct {
	ID          string
	Name        string
	ScriptPath  string
	Schedule    string
	LastRun     time.Time
	NextRun     time.Time
	Status      TaskStatus
	Error       error
	RetryCount  int
	MaxRetries  int
	CronID      cron.EntryID
	Metadata    map[string]string
	ExecutionFn func(context.Context) error
}

// Scheduler manages task scheduling and execution
type Scheduler struct {
	cron       *cron.Cron
	scriptMgr  *scripts.ScriptManager
	tasks      map[string]*Task
	config     *config.SchedConfig
	logger     *zap.Logger
	metrics    *SchedulerMetrics
	workerPool chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

// SchedulerMetrics tracks scheduler performance
type SchedulerMetrics struct {
	TasksScheduled  int64
	TasksCompleted  int64
	TasksFailed     int64
	AverageLatency  time.Duration
	ConcurrentTasks int
	LastUpdate      time.Time
	mu              sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(scriptMgr *scripts.ScriptManager, config *config.SchedConfig, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:       cron.New(cron.WithSeconds()),
		scriptMgr:  scriptMgr,
		tasks:      make(map[string]*Task),
		config:     config,
		logger:     logger,
		metrics:    &SchedulerMetrics{},
		workerPool: make(chan struct{}, config.MaxConcurrent),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	s.logger.Info("Starting scheduler",
		zap.Int("maxConcurrent", s.config.MaxConcurrent))

	// Start metrics collection
	go s.collectMetrics()

	// Start the cron scheduler
	s.cron.Start()

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping scheduler")

	// Cancel context to stop background operations
	s.cancel()

	// Stop accepting new tasks
	ctx := s.cron.Stop()

	// Wait for running tasks to complete
	<-ctx.Done()

	return nil
}

// ScheduleTask adds a new task to the scheduler
func (s *Scheduler) ScheduleTask(task *Task) error {
	if err := s.validateTask(task); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate task
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Create cron schedule
	cronID, err := s.cron.AddFunc(task.Schedule, func() {
		s.executeTask(s.ctx, task)
	})
	if err != nil {
		return fmt.Errorf("scheduling task: %w", err)
	}

	// Update task
	task.CronID = cronID
	task.Status = TaskStatusPending
	task.NextRun = s.cron.Entry(cronID).Next
	s.tasks[task.ID] = task

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.TasksScheduled++
	s.metrics.LastUpdate = time.Now()
	s.metrics.mu.Unlock()

	s.logger.Info("Task scheduled",
		zap.String("taskID", task.ID),
		zap.String("schedule", task.Schedule),
		zap.Time("nextRun", task.NextRun))

	return nil
}

// UnscheduleTask removes a task from the scheduler
func (s *Scheduler) UnscheduleTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	s.cron.Remove(task.CronID)
	delete(s.tasks, taskID)

	s.logger.Info("Task unscheduled",
		zap.String("taskID", taskID))

	return nil
}

// GetTask retrieves a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// ListTasks returns all scheduled tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// Private methods

func (s *Scheduler) executeTask(ctx context.Context, task *Task) {
	// Acquire worker from pool
	select {
	case s.workerPool <- struct{}{}:
		defer func() { <-s.workerPool }()
	case <-ctx.Done():
		return
	}

	start := time.Now()

	s.mu.Lock()
	task.Status = TaskStatusRunning
	task.LastRun = start
	s.mu.Unlock()

	// Execute task
	err := s.runTaskWithRetries(ctx, task)

	s.mu.Lock()
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err
		s.metrics.mu.Lock()
		s.metrics.TasksFailed++
		s.metrics.mu.Unlock()
	} else {
		task.Status = TaskStatusComplete
		task.Error = nil
		s.metrics.mu.Lock()
		s.metrics.TasksCompleted++
		s.metrics.mu.Unlock()
	}
	task.NextRun = s.cron.Entry(task.CronID).Next
	s.mu.Unlock()

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.AverageLatency = (s.metrics.AverageLatency*9 + time.Since(start)) / 10
	s.metrics.LastUpdate = time.Now()
	s.metrics.mu.Unlock()

	s.logger.Info("Task execution completed",
		zap.String("taskID", task.ID),
		zap.Duration("duration", time.Since(start)),
		zap.Error(err))
}

func (s *Scheduler) runTaskWithRetries(ctx context.Context, task *Task) error {
	var lastErr error

	for attempt := 0; attempt <= task.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(s.config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute task function
		if err := task.ExecutionFn(ctx); err != nil {
			lastErr = err
			task.RetryCount = attempt
			s.logger.Warn("Task execution failed",
				zap.String("taskID", task.ID),
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			continue
		}

		return nil
	}

	return fmt.Errorf("task failed after %d retries: %w", task.MaxRetries, lastErr)
}

func (s *Scheduler) validateTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if task.Schedule == "" {
		return fmt.Errorf("task schedule cannot be empty")
	}
	if task.ExecutionFn == nil {
		return fmt.Errorf("task execution function cannot be nil")
	}

	// Validate cron schedule
	if _, err := cron.ParseStandard(task.Schedule); err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	return nil
}

func (s *Scheduler) collectMetrics() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateMetrics()
		}
	}
}

func (s *Scheduler) updateMetrics() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.mu.RLock()
	runningTasks := 0
	for _, task := range s.tasks {
		if task.Status == TaskStatusRunning {
			runningTasks++
		}
	}
	s.mu.RUnlock()

	s.metrics.ConcurrentTasks = runningTasks
	s.metrics.LastUpdate = time.Now()
}

// UpdateTaskSchedule updates the schedule of an existing task
func (s *Scheduler) UpdateTaskSchedule(taskID string, schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Validate new schedule
	if _, err := cron.ParseStandard(schedule); err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	// Remove old schedule
	s.cron.Remove(task.CronID)

	// Add new schedule
	cronID, err := s.cron.AddFunc(schedule, func() {
		s.executeTask(s.ctx, task)
	})
	if err != nil {
		return fmt.Errorf("updating task schedule: %w", err)
	}

	task.Schedule = schedule
	task.CronID = cronID
	task.NextRun = s.cron.Entry(cronID).Next

	s.logger.Info("Task schedule updated",
		zap.String("taskID", taskID),
		zap.String("schedule", schedule),
		zap.Time("nextRun", task.NextRun))

	return nil
}

// GetSchedulerStats returns current scheduler statistics
func (s *Scheduler) GetSchedulerStats() SchedulerStats {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return SchedulerStats{
		TasksScheduled:  s.metrics.TasksScheduled,
		TasksCompleted:  s.metrics.TasksCompleted,
		TasksFailed:     s.metrics.TasksFailed,
		AverageLatency:  s.metrics.AverageLatency,
		ConcurrentTasks: s.metrics.ConcurrentTasks,
		LastUpdate:      s.metrics.LastUpdate,
	}
}

// SchedulerStats represents scheduler statistics
type SchedulerStats struct {
	TasksScheduled  int64
	TasksCompleted  int64
	TasksFailed     int64
	AverageLatency  time.Duration
	ConcurrentTasks int
	LastUpdate      time.Time
}
