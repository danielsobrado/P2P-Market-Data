package scripts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
)

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
	config     *config.ScriptConfig
	logger     *zap.Logger
	metrics    *ExecutorMetrics
	processMap sync.Map // maps script IDs to running processes
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

	// Create result channels
	resultChan := make(chan *ExecutionResult, 1)
	errChan := make(chan error, 1)

	// Start execution
	go func() {
		result, err := e.runScript(ctx, scriptPath, args, env)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for completion or timeout
	select {
	case result := <-resultChan:
		e.updateMetrics(result)
		return result, nil
	case err := <-errChan:
		return nil, err
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
	cmd.SysProcAttr = &syscall.SysProcAttr{
	}

	// Store process for potential cancellation
	e.processMap.Store(scriptPath, cmd.Process)
	defer e.processMap.Delete(scriptPath)

	// Run script
	err := cmd.Run()

	// Prepare result
	result := &ExecutionResult{
		ExitCode:  cmd.ProcessState.ExitCode(),
		Output:    stdout.String(),
		Error:     stderr.String(),
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	// Get resource usage
	result.MemoryUsage = 0 // Memory usage not available on this platform
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

// killScript terminates a running script
func (e *ScriptExecutor) killScript(scriptPath string) {
	if proc, ok := e.processMap.Load(scriptPath); ok {
		if p, ok := proc.(*os.Process); ok {
			p.Kill()
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
		return fmt.Errorf("python path not specified")
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
		return nil, err
	}

	// Parse JSON output
	if err := json.Unmarshal([]byte(result.Output), outputType); err != nil {
		return nil, fmt.Errorf("parsing script output: %w", err)
	}

	return result, nil
}
