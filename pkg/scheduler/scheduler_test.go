package scheduler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/scripts"
)

func setupTestScheduler(t *testing.T) *Scheduler {
	logger := zaptest.NewLogger(t)
	scriptMgr := &scripts.Manager{} // Mock script manager
	cfg := &config.SchedConfig{
		MaxConcurrent: 5,
		RetryDelay:    time.Second,
	}

	scheduler := NewScheduler(scriptMgr, cfg, logger)
	require.NoError(t, scheduler.Start())

	return scheduler
}

func TestScheduleTask(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	t.Run("ValidTask", func(t *testing.T) {
		task := &Task{
			ID:         "test-task-1",
			Name:       "Test Task",
			Schedule:   "*/5 * * * * *", // Every 5 seconds
			MaxRetries: 3,
			ExecutionFn: func(ctx context.Context) error {
				return nil
			},
		}

		err := scheduler.ScheduleTask(task)
		require.NoError(t, err)

		// Verify task was scheduled
		scheduledTask, err := scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, scheduledTask.ID)
		assert.Equal(t, TaskStatusPending, scheduledTask.Status)
	})

	t.Run("InvalidSchedule", func(t *testing.T) {
		task := &Task{
			ID:       "test-task-2",
			Schedule: "invalid",
			ExecutionFn: func(ctx context.Context) error {
				return nil
			},
		}

		err := scheduler.ScheduleTask(task)
		assert.Error(t, err)
	})

	t.Run("DuplicateTask", func(t *testing.T) {
		task := &Task{
			ID:       "test-task-3",
			Schedule: "* * * * * *",
			ExecutionFn: func(ctx context.Context) error {
				return nil
			},
		}

		err := scheduler.ScheduleTask(task)
		require.NoError(t, err)

		err = scheduler.ScheduleTask(task)
		assert.Error(t, err)
	})
}

func TestTaskExecution(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	t.Run("SuccessfulExecution", func(t *testing.T) {
		executed := make(chan bool, 1)
		task := &Task{
			ID:       "test-task-4",
			Schedule: "* * * * * *", // Every second
			ExecutionFn: func(ctx context.Context) error {
				executed <- true
				return nil
			},
		}

		err := scheduler.ScheduleTask(task)
		require.NoError(t, err)

		// Wait for execution
		select {
		case <-executed:
			// Task executed successfully
		case <-time.After(2 * time.Second):
			t.Fatal("Task execution timeout")
		}

		// Verify task status
		scheduledTask, err := scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TaskStatusComplete, scheduledTask.Status)
		assert.Nil(t, scheduledTask.Error)
	})

	t.Run("FailedExecution", func(t *testing.T) {
		expectedErr := errors.New("execution failed")
		task := &Task{
			ID:         "test-task-5",
			Schedule:   "* * * * * *",
			MaxRetries: 1,
			ExecutionFn: func(ctx context.Context) error {
				return expectedErr
			},
		}

		err := scheduler.ScheduleTask(task)
		require.NoError(t, err)

		// Wait for execution and retries
		time.Sleep(3 * time.Second)

		// Verify task status
		scheduledTask, err := scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TaskStatusFailed, scheduledTask.Status)
		assert.ErrorIs(t, scheduledTask.Error, expectedErr)
		assert.Equal(t, 1, scheduledTask.RetryCount)
	})

	t.Run("ConcurrentExecution", func(t *testing.T) {
		const numTasks = 10
		executionCount := make(chan int, numTasks)
		completedTasks := 0

		// Create multiple tasks
		for i := 0; i < numTasks; i++ {
			task := &Task{
				ID:       fmt.Sprintf("concurrent-task-%d", i),
				Schedule: "* * * * * *",
				ExecutionFn: func(ctx context.Context) error {
					time.Sleep(100 * time.Millisecond) // Simulate work
					executionCount <- 1
					return nil
				},
			}
			err := scheduler.ScheduleTask(task)
			require.NoError(t, err)
		}

		// Wait for executions
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-executionCount:
				completedTasks++
				if completedTasks == numTasks {
					return
				}
			case <-timeout:
				t.Fatalf("Only completed %d out of %d tasks", completedTasks, numTasks)
			}
		}
	})
}

func TestTaskCancellation(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	t.Run("CancelRunningTask", func(t *testing.T) {
		started := make(chan bool)
		completed := make(chan bool)

		task := &Task{
			ID:       "test-task-6",
			Schedule: "* * * * * *",
			ExecutionFn: func(ctx context.Context) error {
				started <- true
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					completed <- true
					return nil
				}
			},
		}

		err := scheduler.ScheduleTask(task)
		require.NoError(t, err)

		// Wait for task to start
		<-started

		// Cancel the scheduler
		err = scheduler.Stop()
		require.NoError(t, err)

		// Verify task was cancelled
		select {
		case <-completed:
			t.Fatal("Task should have been cancelled")
		case <-time.After(100 * time.Millisecond):
			// Task was successfully cancelled
		}
	})
}

func TestSchedulerMetrics(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	// Schedule and execute some tasks
	successTask := &Task{
		ID:       "metrics-task-1",
		Schedule: "* * * * * *",
		ExecutionFn: func(ctx context.Context) error {
			return nil
		},
	}

	failTask := &Task{
		ID:       "metrics-task-2",
		Schedule: "* * * * * *",
		ExecutionFn: func(ctx context.Context) error {
			return errors.New("task failed")
		},
	}

	require.NoError(t, scheduler.ScheduleTask(successTask))
	require.NoError(t, scheduler.ScheduleTask(failTask))

	// Wait for executions
	time.Sleep(2 * time.Second)

	// Check metrics
	stats := scheduler.GetSchedulerStats()
	assert.Equal(t, int64(2), stats.TasksScheduled)
	assert.GreaterOrEqual(t, stats.TasksCompleted, int64(1))
	assert.GreaterOrEqual(t, stats.TasksFailed, int64(1))
	assert.NotZero(t, stats.AverageLatency)
}

func TestScheduleUpdate(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	executionTimes := make(chan time.Time, 2)
	task := &Task{
		ID:       "update-schedule-task",
		Schedule: "*/5 * * * * *", // Every 5 seconds
		ExecutionFn: func(ctx context.Context) error {
			executionTimes <- time.Now()
			return nil
		},
	}

	// Schedule initial task
	require.NoError(t, scheduler.ScheduleTask(task))

	// Update schedule to run every second
	err := scheduler.UpdateTaskSchedule(task.ID, "* * * * * *")
	require.NoError(t, err)

	// Wait for executions
	firstExecution := <-executionTimes
	secondExecution := <-executionTimes

	// Verify the interval between executions is closer to 1 second than 5 seconds
	interval := secondExecution.Sub(firstExecution)
	assert.Less(t, interval, 2*time.Second)
}

func TestTaskRetryBehavior(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	attempts := 0
	task := &Task{
		ID:         "retry-task",
		Schedule:   "* * * * * *",
		MaxRetries: 2,
		ExecutionFn: func(ctx context.Context) error {
			attempts++
			if attempts <= 2 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	require.NoError(t, scheduler.ScheduleTask(task))

	// Wait for retries
	time.Sleep(4 * time.Second)

	scheduledTask, err := scheduler.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusComplete, scheduledTask.Status)
	assert.Equal(t, 2, scheduledTask.RetryCount)
}

func TestSchedulerGracefulShutdown(t *testing.T) {
	scheduler := setupTestScheduler(t)

	// Add a long-running task
	taskCompleted := make(chan bool)
	task := &Task{
		ID:       "shutdown-task",
		Schedule: "* * * * * *",
		ExecutionFn: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				taskCompleted <- true
				return nil
			}
		},
	}

	require.NoError(t, scheduler.ScheduleTask(task))

	// Wait for task to start
	time.Sleep(100 * time.Millisecond)

	// Start shutdown
	shutdownComplete := make(chan bool)
	go func() {
		err := scheduler.Stop()
		require.NoError(t, err)
		shutdownComplete <- true
	}()

	// Verify shutdown behavior
	select {
	case <-shutdownComplete:
		// Shutdown completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Scheduler shutdown timeout")
	}
}

func TestSchedulerRecovery(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	panicCount := 0
	task := &Task{
		ID:         "recovery-task",
		Schedule:   "* * * * * *",
		MaxRetries: 1,
		ExecutionFn: func(ctx context.Context) error {
			panicCount++
			if panicCount == 1 {
				panic("unexpected panic")
			}
			return nil
		},
	}

	require.NoError(t, scheduler.ScheduleTask(task))

	// Wait for recovery
	time.Sleep(3 * time.Second)

	scheduledTask, err := scheduler.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusComplete, scheduledTask.Status)
	assert.Equal(t, 1, scheduledTask.RetryCount)
}
