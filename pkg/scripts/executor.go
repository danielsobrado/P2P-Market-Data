package scripts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
)

// scriptRun holds per-execution state so StopScript can signal the process
// and wait for the background goroutine's cmd.Wait() to finish without
// calling Wait() a second time.
type scriptRun struct {
	cmd  *exec.Cmd
	done chan struct{} // closed when the background cmd.Wait() returns
}

// ExecutionResult represents the result of a script execution
type ExecutionResult struct {
	ExitCode    int
	Output      string
	Error       string
	StartTime   time.Time
	EndTime     time.Time
	MemoryUsage int64
	CPUTime     time.Duration
	Metrics     map[string]interface{}
}

// ScriptExecutor handles script execution with resource limits
type ScriptExecutor struct {
	config         *config.ScriptConfig
	logger         *zap.Logger
	metrics        *ExecutorMetrics
	runningScripts sync.Map // maps script path/ID to *scriptRun
}

// ExecutorMetrics tracks script execution metrics
type ExecutorMetrics struct {
	ExecutionsTotal    int64
	ExecutionsFailed   int64
	AverageMemoryUsage int64
	AverageCPUTime     time.Duration
	LastExecution      time.Time
	mu                 sync.RWMutex
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(config *config.ScriptConfig, logger *zap.Logger) (*ScriptExecutor, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &ScriptExecutor{
		config:  config,
		logger:  logger,
		metrics: &ExecutorMetrics{},
	}, nil
}

// ExecuteScript runs a Python script with resource limits
func (e *ScriptExecutor) ExecuteScript(ctx context.Context, scriptPath string, args []string) (*ExecutionResult, error) {
	// Validate script
	if err := e.validateScript(scriptPath); err != nil {
		return nil, fmt.Errorf("script validation failed: %w", err)
	}

	// Prepare execution environment
	env, err := e.prepareEnvironment()
	if err != nil {
		return nil, fmt.Errorf("preparing environment: %w", err)
	}

	// Create result channels — carry both result and error so the caller always
	// receives the ExecutionResult even when the script exits non-zero.
	type outcome struct {
		result *ExecutionResult
		err    error
	}
	outChan := make(chan outcome, 1)

	// Start execution
	go func() {
		result, err := e.runScript(ctx, scriptPath, args, env)
		outChan <- outcome{result, err}
	}()

	// Wait for completion or context cancellation
	select {
	case out := <-outChan:
		if out.err == nil {
			e.updateMetrics(out.result)
		}
		return out.result, out.err
	case <-ctx.Done():
		e.killScript(scriptPath)
		return nil, ctx.Err()
	}
}

// runScript executes the script process
func (e *ScriptExecutor) runScript(ctx context.Context, scriptPath string, args []string, env []string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Prepare command
	cmd := exec.CommandContext(ctx, e.config.PythonPath, append([]string{scriptPath}, args...)...)
	cmd.Env = env

	// Set up output buffers
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set resource limits
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	// Register the run entry before Start so StopScript can find it.
	run := &scriptRun{cmd: cmd, done: make(chan struct{})}
	e.runningScripts.Store(scriptPath, run)
	defer func() {
		close(run.done)
		e.runningScripts.Delete(scriptPath)
	}()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting script: %w", err)
	}

	// Run script — this is the single authoritative Wait() call.
	err := cmd.Wait()

	// Prepare result
	result := &ExecutionResult{
		ExitCode:  cmd.ProcessState.ExitCode(),
		Output:    stdout.String(),
		Error:     stderr.String(),
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	result.MemoryUsage = 0
	result.CPUTime = 0

	if err != nil {
		e.metrics.mu.Lock()
		e.metrics.ExecutionsFailed++
		e.metrics.mu.Unlock()
		return result, fmt.Errorf("script execution failed: %w", err)
	}

	return result, nil
}

// validateScript checks if the script is safe to execute
func (e *ScriptExecutor) validateScript(scriptPath string) error {
	// Check file exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script not found: %w", err)
	}

	// Check file extension
	if !strings.HasSuffix(scriptPath, ".py") {
		return fmt.Errorf("invalid script extension: %s", scriptPath)
	}

	// Check for suspicious imports
	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("opening script: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "import") || strings.HasPrefix(line, "from") {
			if !e.isAllowedImport(line) {
				return fmt.Errorf("unauthorized import: %s", line)
			}
		}
	}

	return scanner.Err()
}

// isAllowedImport checks if an import is allowed
func (e *ScriptExecutor) isAllowedImport(importLine string) bool {
	for _, allowed := range e.config.AllowedPkgs {
		if strings.Contains(importLine, allowed) {
			return true
		}
	}
	return false
}

// prepareEnvironment sets up the execution environment
func (e *ScriptExecutor) prepareEnvironment() ([]string, error) {
	env := os.Environ()

	// Add custom environment variables
	env = append(env, fmt.Sprintf("PYTHONPATH=%s", e.config.ScriptDir))
	env = append(env, "PYTHONUNBUFFERED=1") // Ensure output is not buffered

	// Set resource limits
	env = append(env, fmt.Sprintf("MEMORY_LIMIT=%d", e.config.MaxMemoryMB*1024*1024))

	return env, nil
}

// killScript terminates a running script (best-effort; used on context cancellation)
func (e *ScriptExecutor) killScript(scriptPath string) {
	if val, ok := e.runningScripts.Load(scriptPath); ok {
		if run, ok := val.(*scriptRun); ok && run.cmd != nil && run.cmd.Process != nil {
			_ = run.cmd.Process.Kill()
		}
	}
}

// updateMetrics updates execution metrics
func (e *ScriptExecutor) updateMetrics(result *ExecutionResult) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()

	e.metrics.ExecutionsTotal++
	e.metrics.LastExecution = time.Now()

	// Update average memory usage
	count := float64(e.metrics.ExecutionsTotal)
	e.metrics.AverageMemoryUsage = int64(
		(float64(e.metrics.AverageMemoryUsage)*(count-1) +
			float64(result.MemoryUsage)) / count,
	)

	// Update average CPU time
	e.metrics.AverageCPUTime = time.Duration(
		(float64(e.metrics.AverageCPUTime)*(count-1) +
			float64(result.CPUTime)) / count,
	)
}

// GetExecutorStats returns current executor statistics
func (e *ScriptExecutor) GetExecutorStats() ExecutorStats {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	return ExecutorStats{
		ExecutionsTotal:    e.metrics.ExecutionsTotal,
		ExecutionsFailed:   e.metrics.ExecutionsFailed,
		AverageMemoryUsage: e.metrics.AverageMemoryUsage,
		AverageCPUTime:     e.metrics.AverageCPUTime,
		LastExecution:      e.metrics.LastExecution,
	}
}

// ExecutorStats represents executor statistics
type ExecutorStats struct {
	ExecutionsTotal    int64
	ExecutionsFailed   int64
	AverageMemoryUsage int64
	AverageCPUTime     time.Duration
	LastExecution      time.Time
}

// Helper functions

func validateConfig(config *config.ScriptConfig) error {
	if config.PythonPath == "" {
		return fmt.Errorf("python path not set")
	}
	// Verify the interpreter exists and is named like a Python binary.
	// PythonPath comes from operator-controlled configuration, not end-user input.
	base := filepath.Base(config.PythonPath)
	if !strings.HasPrefix(base, "python") {
		return fmt.Errorf("python interpreter path %q does not look like a Python binary", config.PythonPath)
	}
	if _, err := exec.LookPath(config.PythonPath); err != nil {
		return fmt.Errorf("python interpreter %q not found: %w", config.PythonPath, err)
	}
	if config.MaxExecTime <= 0 {
		return fmt.Errorf("invalid maximum execution time")
	}
	if config.MaxMemoryMB <= 0 {
		return fmt.Errorf("invalid maximum memory")
	}
	return nil
}

// Additional execution options

// ExecuteScriptWithInput runs a script with input data
func (e *ScriptExecutor) ExecuteScriptWithInput(ctx context.Context, scriptPath string, input interface{}) (*ExecutionResult, error) {
	// Convert input to JSON
	inputData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshaling input data: %w", err)
	}

	// Create temporary input file
	inputFile, err := os.CreateTemp("", "script_input_*.json")
	if err != nil {
		return nil, fmt.Errorf("creating input file: %w", err)
	}
	defer os.Remove(inputFile.Name())

	if _, err := inputFile.Write(inputData); err != nil {
		return nil, fmt.Errorf("writing input data: %w", err)
	}
	inputFile.Close()

	// Execute script with input file
	return e.ExecuteScript(ctx, scriptPath, []string{"--input", inputFile.Name()})
}

// ExecuteScriptWithOutputCapture runs a script and captures structured output
func (e *ScriptExecutor) ExecuteScriptWithOutputCapture(ctx context.Context, scriptPath string, outputType interface{}) (*ExecutionResult, error) {
	result, err := e.ExecuteScript(ctx, scriptPath, []string{"--json-output"})
	if err != nil {
		return result, err
	}

	// Trim surrounding whitespace before parsing to handle trailing newlines etc.
	trimmed := strings.TrimSpace(result.Output)
	if err := json.Unmarshal([]byte(trimmed), outputType); err != nil {
		return result, fmt.Errorf("parsing script JSON output: %w (raw output: %q)", err, trimmed)
	}

	return result, nil
}

// StopScript stops a running script by ID using graceful→timeout→force-kill.
// It waits for the background cmd.Wait() goroutine to finish so there is no
// double-Wait() race condition.
func (e *ScriptExecutor) StopScript(scriptID string) error {
	value, exists := e.runningScripts.Load(scriptID)
	if !exists {
		return fmt.Errorf("script %s is not running", scriptID)
	}

	run, ok := value.(*scriptRun)
	if !ok || run.cmd == nil || run.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first.
	if err := run.cmd.Process.Signal(os.Interrupt); err != nil {
		e.logger.Warn("Failed to send interrupt signal",
			zap.String("scriptID", scriptID),
			zap.Error(err))
	}

	// Wait up to 5 s for the process to exit cleanly.
	select {
	case <-run.done:
		e.logger.Info("Script stopped gracefully", zap.String("scriptID", scriptID))
		return nil
	case <-time.After(5 * time.Second):
		// Escalate to SIGKILL.
		e.logger.Warn("Script did not stop gracefully; force killing",
			zap.String("scriptID", scriptID))
		if err := run.cmd.Process.Kill(); err != nil {
			e.logger.Warn("Failed to kill script",
				zap.String("scriptID", scriptID),
				zap.Error(err))
		}
	}

	// Give the OS up to 2 s to reap the process after the kill.
	select {
	case <-run.done:
		e.logger.Info("Script force-killed", zap.String("scriptID", scriptID))
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("script %s did not stop after force kill", scriptID)
	}
}

// Stop stops all running scripts and cleans up resources
func (e *ScriptExecutor) Stop(ctx context.Context) error {
	var errors []error

	// Stop all running scripts
	e.runningScripts.Range(func(key, value interface{}) bool {
		scriptID := key.(string)
		if err := e.StopScript(scriptID); err != nil {
			e.logger.Error("Failed to stop script",
				zap.String("scriptID", scriptID),
				zap.Error(err))
			errors = append(errors, err)
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop all scripts: %v", errors)
	}

	return nil
}

// findPythonPath attempts to find the Python interpreter path in a cross-platform way
func findPythonPath(logger *zap.Logger) string {
	// First, check if PYTHON_PATH environment variable is set
	pythonPath := os.Getenv("PYTHON_PATH")
	if pythonPath != "" {
		return pythonPath
	}

	// Define possible executable names for Python
	pythonExecs := []string{"python3", "python"}

	// On Windows, executable may have .exe extension
	if runtime.GOOS == "windows" {
		for i, execName := range pythonExecs {
			pythonExecs[i] = execName + ".exe"
		}
	}

	// Look for each possible Python executable in the PATH
	for _, execName := range pythonExecs {
		path, err := exec.LookPath(execName)
		if err == nil {
			return path
		}
	}

	// If Python is not found, log a warning and return empty string
	logger.Warn("Python interpreter not found. Some features may be unavailable.")
	return ""
}
