package scripts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"p2p_market_data/pkg/config"
)

func setupTestEnvironment(t *testing.T) (*ScriptManager, string) {
	tempDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	cfg := &config.ScriptConfig{
		ScriptDir:   tempDir,
		PythonPath:  "python3",
		MaxExecTime: time.Minute,
		MaxMemoryMB: 512,
		AllowedPkgs: []string{"pandas", "numpy"},
	}

	manager, err := NewScriptManager(cfg, logger)
	require.NoError(t, err)

	return manager, tempDir
}

func TestScriptExecution(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("ValidScript", func(t *testing.T) {
		script := []byte(`
print("Hello, World!")
`)
		metadata := &ScriptMetadata{
			Name:        "test.py",
			Description: "Test script",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Output, "Hello, World!")
	})

	t.Run("InvalidScript", func(t *testing.T) {
		script := []byte(`
import unauthorized_package
`)
		metadata := &ScriptMetadata{
			Name: "invalid.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		assert.Error(t, err)
	})

	t.Run("ScriptTimeout", func(t *testing.T) {
		script := []byte(`
import time
time.sleep(10)
`)
		metadata := &ScriptMetadata{
			Name: "timeout.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err = manager.ExecuteScript(ctx, metadata.ID, nil)
		assert.Error(t, err)
	})
}

func TestScriptManagement(t *testing.T) {
	manager, tempDir := setupTestEnvironment(t)

	t.Run("AddAndRetrieve", func(t *testing.T) {
		script := []byte(`print("Test")`)
		metadata := &ScriptMetadata{
			Name:        "test1.py",
			Description: "Test script 1",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		retrieved, err := manager.GetScript(metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, metadata.Name, retrieved.Name)
	})

	t.Run("UpdateScript", func(t *testing.T) {
		script := []byte(`print("Original")`)
		metadata := &ScriptMetadata{
			Name: "test2.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		newContent := []byte(`print("Updated")`)
		err = manager.UpdateScript(metadata.ID, newContent)
		require.NoError(t, err)

		// Verify the update
		content, err := os.ReadFile(filepath.Join(tempDir, metadata.Name))
		require.NoError(t, err)
		assert.Equal(t, newContent, content)
	})

	t.Run("DeleteScript", func(t *testing.T) {
		script := []byte(`print("Delete me")`)
		metadata := &ScriptMetadata{
			Name: "test3.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		err = manager.DeleteScript(metadata.ID)
		require.NoError(t, err)

		_, err = manager.GetScript(metadata.ID)
		assert.Error(t, err)

		// Verify file is removed
		_, err = os.Stat(filepath.Join(tempDir, metadata.Name))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("ListScripts", func(t *testing.T) {
		// Add multiple scripts
		for i := 1; i <= 3; i++ {
			script := []byte(fmt.Sprintf(`print("Script %d")`, i))
			metadata := &ScriptMetadata{
				Name: fmt.Sprintf("list_test_%d.py", i),
			}
			err := manager.AddScript(metadata.Name, script, metadata)
			require.NoError(t, err)
		}

		scripts := manager.ListScripts()
		assert.GreaterOrEqual(t, len(scripts), 3)
	})
}

func TestScriptIntegrity(t *testing.T) {
	manager, tempDir := setupTestEnvironment(t)

	t.Run("VerifyIntegrity", func(t *testing.T) {
		script := []byte(`print("Original")`)
		metadata := &ScriptMetadata{
			Name: "integrity_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		// Verify unmodified script
		intact, err := manager.VerifyScriptIntegrity(metadata.ID)
		require.NoError(t, err)
		assert.True(t, intact)

		// Modify script directly
		err = os.WriteFile(filepath.Join(tempDir, metadata.Name), []byte(`print("Modified")`), 0644)
		require.NoError(t, err)

		// Verify modified script
		intact, err = manager.VerifyScriptIntegrity(metadata.ID)
		require.NoError(t, err)
		assert.False(t, intact)
	})
	t.Run("BackupAndHistory", func(t *testing.T) {
		script := []byte(`print("Version 1")`)
		metadata := &ScriptMetadata{
			Name: "backup_test.py",
		}

		// Add initial version
		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		// Create backup
		err = manager.BackupScript(metadata.ID)
		require.NoError(t, err)

		// Update script
		newContent := []byte(`print("Version 2")`)
		err = manager.UpdateScript(metadata.ID, newContent)
		require.NoError(t, err)

		// Create another backup
		err = manager.BackupScript(metadata.ID)
		require.NoError(t, err)

		// Get history
		history, err := manager.GetScriptHistory(metadata.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 2)
	})

	t.Run("DependencyValidation", func(t *testing.T) {
		script := []byte(`
import pandas as pd
import numpy as np

data = pd.DataFrame({'A': np.random.rand(5)})
print(data)
`)
		metadata := &ScriptMetadata{
			Name:         "deps_test.py",
			Dependencies: []string{"pandas", "numpy"},
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		err = manager.ValidateDependencies(metadata.ID)
		require.NoError(t, err)
	})
}

func TestScriptExecutionLimits(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("MemoryLimit", func(t *testing.T) {
		script := []byte(`
import numpy as np
# Attempt to allocate a large array
data = np.zeros((1024, 1024, 1024))  # ~8GB
`)
		metadata := &ScriptMetadata{
			Name: "memory_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		assert.Error(t, err)
		assert.NotEqual(t, 0, result.ExitCode)
	})

	t.Run("CPUTimeTracking", func(t *testing.T) {
		script := []byte(`
import time
start = time.time()
# CPU-intensive operation
for i in range(1000000):
    _ = i ** 2
print(f"Execution time: {time.time() - start}")
`)
		metadata := &ScriptMetadata{
			Name: "cpu_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		require.NoError(t, err)
		assert.Greater(t, result.CPUTime, time.Duration(0))
	})
}

func TestConcurrentExecution(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("MultipleConcurrentScripts", func(t *testing.T) {
		script := []byte(`
import time
import random
sleep_time = random.uniform(0.1, 0.5)
time.sleep(sleep_time)
print(f"Slept for {sleep_time} seconds")
`)
		metadata := &ScriptMetadata{
			Name: "concurrent_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		// Execute multiple instances concurrently
		var wg sync.WaitGroup
		results := make(chan *ExecutionResult, 5)
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}()
		}

		wg.Wait()
		close(results)
		close(errors)

		// Check results
		successCount := 0
		for result := range results {
			assert.Equal(t, 0, result.ExitCode)
			successCount++
		}

		// Check errors
		for err := range errors {
			assert.NoError(t, err)
		}

		assert.Equal(t, 5, successCount)
	})
}

func TestScriptExecutionWithInput(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("ExecuteWithJSONInput", func(t *testing.T) {
		script := []byte(`
import json
import sys

with open(sys.argv[1]) as f:
    data = json.load(f)
print(f"Processed input: {data['value'] * 2}")
`)
		metadata := &ScriptMetadata{
			Name: "input_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		input := map[string]int{"value": 21}
		result, err := manager.Executor.ExecuteScriptWithInput(context.Background(),
			filepath.Join(manager.Config.ScriptDir, metadata.Name), input)

		require.NoError(t, err)
		assert.Contains(t, result.Output, "Processed input: 42")
	})

	t.Run("ExecuteWithStructuredOutput", func(t *testing.T) {
		script := []byte(`
import json
result = {"calculated": 42, "message": "success"}
print(json.dumps(result))
`)
		metadata := &ScriptMetadata{
			Name: "output_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		var output struct {
			Calculated int    `json:"calculated"`
			Message    string `json:"message"`
		}

		result, err := manager.Executor.ExecuteScriptWithOutputCapture(context.Background(),
			filepath.Join(manager.Config.ScriptDir, metadata.Name), &output)

		require.NoError(t, err)
		assert.Equal(t, 42, output.Calculated)
		assert.Equal(t, "success", output.Message)
		assert.Equal(t, 0, result.ExitCode)
	})
}

func TestScriptMetrics(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("ExecutorMetrics", func(t *testing.T) {
		script := []byte(`print("Test metrics")`)
		metadata := &ScriptMetadata{
			Name: "metrics_test.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		// Execute script multiple times
		for i := 0; i < 3; i++ {
			_, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
			require.NoError(t, err)
		}

		stats := manager.Executor.GetExecutorStats()
		assert.Equal(t, int64(3), stats.ExecutionsTotal)
		assert.Equal(t, int64(0), stats.ExecutionsFailed)
		assert.Greater(t, stats.AverageMemoryUsage, int64(0))
		assert.Greater(t, stats.AverageCPUTime, time.Duration(0))
	})
}

func TestScriptErrorHandling(t *testing.T) {
	manager, _ := setupTestEnvironment(t)

	t.Run("SyntaxError", func(t *testing.T) {
		script := []byte(`
print("Unclosed string
`)
		metadata := &ScriptMetadata{
			Name: "syntax_error.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		assert.Error(t, err)
		assert.NotEqual(t, 0, result.ExitCode)
		assert.Contains(t, result.Error, "SyntaxError")
	})

	t.Run("RuntimeError", func(t *testing.T) {
		script := []byte(`
def divide_by_zero():
    return 1 / 0

divide_by_zero()
`)
		metadata := &ScriptMetadata{
			Name: "runtime_error.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		assert.Error(t, err)
		assert.NotEqual(t, 0, result.ExitCode)
		assert.Contains(t, result.Error, "ZeroDivisionError")
	})

	t.Run("ImportError", func(t *testing.T) {
		script := []byte(`
import non_existent_package
`)
		metadata := &ScriptMetadata{
			Name: "import_error.py",
		}

		err := manager.AddScript(metadata.Name, script, metadata)
		require.NoError(t, err)

		result, err := manager.ExecuteScript(context.Background(), metadata.ID, nil)
		assert.Error(t, err)
		assert.NotEqual(t, 0, result.ExitCode)
		assert.Contains(t, result.Error, "ImportError")
	})
}
